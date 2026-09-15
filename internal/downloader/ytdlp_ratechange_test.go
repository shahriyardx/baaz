package downloader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeYtdlp installs a stand-in yt-dlp on PATH that appends its argv to a log
// and streams progress lines until it is killed. The Nth run exits cleanly so
// the job can finish.
func fakeYtdlp(t *testing.T, finishOnRun int) (dir, log string) {
	t.Helper()
	dir = t.TempDir()
	log = filepath.Join(dir, "argv.log")
	script := `#!/bin/sh
# PATH is inherited, so wc/sleep resolve normally.
echo "$@" >> "` + log + `"
runs=$(wc -l < "` + log + `")
echo '[download] Destination: /tmp/fake.mp4'
i=0
while [ $i -lt 80 ]; do
  echo "[download]  $i.0% of 100.00MiB at 1.00MiB/s ETA 00:10"
  if [ "$runs" -ge ` + itoa(finishOnRun) + ` ] && [ $i -ge 2 ]; then
    echo '[Merger] Merging formats into "/tmp/fake.mp4"'
    exit 0
  fi
  i=$((i+1))
  sleep 0.1
done
exit 0
`
	p := filepath.Join(dir, "yt-dlp")
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, log
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	panic("small numbers only")
}

func runs(t *testing.T, log string) []string {
	t.Helper()
	b, err := os.ReadFile(log)
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// The bug: changing the speed limit during a media download did nothing until
// the user paused and resumed, because --limit-rate is read once at exec.
func TestSpeedLimitChangeRestartsYtdlpWithTheNewCap(t *testing.T) {
	dir, log := fakeYtdlp(t, 2)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	e := NewEngine(1, 1<<30)
	e.Limiter.SetRate(1000 << 10) // 1000 KB/s
	j := &Job{URL: "https://youtube.com/watch?v=x", Dir: t.TempDir(), Kind: KindMedia}

	done := make(chan error, 1)
	go func() { done <- e.runYtdlp(context.Background(), j) }()

	time.Sleep(600 * time.Millisecond)
	e.Limiter.SetRate(500 << 10) // the user drops it to 500 KB/s mid-download

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("job failed: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("job never finished")
	}

	got := runs(t, log)
	if len(got) != 2 {
		t.Fatalf("expected 2 yt-dlp runs (one restart), got %d:\n%s", len(got), strings.Join(got, "\n"))
	}
	if !strings.Contains(got[0], "--limit-rate 1000K") {
		t.Errorf("first run did not carry the original cap: %s", got[0])
	}
	if !strings.Contains(got[1], "--limit-rate 500K") {
		t.Errorf("restart did not carry the new cap: %s", got[1])
	}
	if !strings.Contains(got[1], "-c") {
		t.Errorf("restart must resume, not start over: %s", got[1])
	}
}

// A stepper emits a change per click; each must not cost its own restart.
func TestBurstOfChangesCostsOneRestart(t *testing.T) {
	dir, log := fakeYtdlp(t, 2)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	e := NewEngine(1, 1<<30)
	e.Limiter.SetRate(1000 << 10)
	j := &Job{URL: "https://youtube.com/watch?v=x", Dir: t.TempDir(), Kind: KindMedia}

	done := make(chan error, 1)
	go func() { done <- e.runYtdlp(context.Background(), j) }()

	time.Sleep(400 * time.Millisecond)
	for _, kb := range []int64{900, 800, 700, 600, 500} {
		e.Limiter.SetRate(kb << 10)
		time.Sleep(80 * time.Millisecond)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("job failed: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("job never finished")
	}

	got := runs(t, log)
	if len(got) != 2 {
		t.Fatalf("5 clicks should collapse into 1 restart, got %d runs:\n%s", len(got), strings.Join(got, "\n"))
	}
	if !strings.Contains(got[1], "--limit-rate 500K") {
		t.Errorf("restart should use the settled value, not an intermediate one: %s", got[1])
	}
}

// Landing back on the starting value is not a change worth restarting for.
func TestChangeThatSettlesBackIsIgnored(t *testing.T) {
	dir, log := fakeYtdlp(t, 1)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	e := NewEngine(1, 1<<30)
	e.Limiter.SetRate(1000 << 10)
	j := &Job{URL: "https://youtube.com/watch?v=x", Dir: t.TempDir(), Kind: KindMedia}

	done := make(chan error, 1)
	go func() { done <- e.runYtdlp(context.Background(), j) }()
	time.Sleep(300 * time.Millisecond)
	e.Limiter.SetRate(500 << 10)
	time.Sleep(50 * time.Millisecond)
	e.Limiter.SetRate(1000 << 10) // undone before it settled

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("job never finished")
	}
	if got := runs(t, log); len(got) != 1 {
		t.Errorf("expected no restart, got %d runs:\n%s", len(got), strings.Join(got, "\n"))
	}
}

// Lifting the cap entirely must also apply at once.
func TestLiftingTheCapRestartsWithoutLimitRate(t *testing.T) {
	dir, log := fakeYtdlp(t, 2)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	e := NewEngine(1, 1<<30)
	e.Limiter.SetRate(500 << 10)
	j := &Job{URL: "https://youtube.com/watch?v=x", Dir: t.TempDir(), Kind: KindMedia}

	done := make(chan error, 1)
	go func() { done <- e.runYtdlp(context.Background(), j) }()
	time.Sleep(600 * time.Millisecond)
	e.Limiter.SetRate(0) // unlimited

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("job never finished")
	}
	got := runs(t, log)
	if len(got) != 2 {
		t.Fatalf("expected a restart, got %d runs:\n%s", len(got), strings.Join(got, "\n"))
	}
	if strings.Contains(got[1], "--limit-rate") {
		t.Errorf("uncapped run should carry no --limit-rate: %s", got[1])
	}
}

// Once the bytes are down and ffmpeg is merging, a cap change has nothing
// left to apply to — restarting would only discard the merge.
func TestNoRestartOnceMergingHasStarted(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "argv.log")
	script := `#!/bin/sh
echo "$@" >> "` + log + `"
echo '[download] Destination: /tmp/fake.mp4'
echo '[download] 100.0% of 100.00MiB at 1.00MiB/s ETA 00:00'
echo '[Merger] Merging formats into "/tmp/fake.mp4"'
sleep 2
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "yt-dlp"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	e := NewEngine(1, 1<<30)
	e.Limiter.SetRate(1000 << 10)
	j := &Job{URL: "https://youtube.com/watch?v=x", Dir: t.TempDir(), Kind: KindMedia}

	done := make(chan error, 1)
	go func() { done <- e.runYtdlp(context.Background(), j) }()

	time.Sleep(500 * time.Millisecond) // merging by now
	e.Limiter.SetRate(500 << 10)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("job failed: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("job never finished")
	}
	if got := runs(t, log); len(got) != 1 {
		t.Errorf("merge must not be restarted, got %d runs:\n%s", len(got), strings.Join(got, "\n"))
	}
}

// yt-dlp prints no progress lines at all when the file is already on disk,
// which left a finished video showing "0B" beside a green tick.
func TestFinishedMediaTakesItsSizeFromTheFile(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	target := filepath.Join(out, "already-there.mp4")
	if err := os.WriteFile(target, make([]byte, 1234567), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"echo '[download] " + target + " has already been downloaded'\n" +
		"echo '[download] Destination: " + target + "'\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "yt-dlp"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	e := NewEngine(1, 1<<30)
	j := &Job{URL: "https://youtube.com/watch?v=x", Dir: out, Kind: KindMedia}
	if err := e.runYtdlp(context.Background(), j); err != nil {
		t.Fatalf("job failed: %v", err)
	}
	if j.Total != 1234567 {
		t.Errorf("Total = %d, want the file's 1234567 bytes", j.Total)
	}
	if got := j.Done(); got != 1234567 {
		t.Errorf("Done() = %d, want 1234567 — progress must match the size", got)
	}
}

// Where yt-dlp does report progress, the finished file is still the honest
// size: the progress figure is the largest single stream, not the merge.
func TestFinishedMediaPrefersTheFileOverProgressTotals(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	target := filepath.Join(out, "merged.mp4")
	if err := os.WriteFile(target, make([]byte, 9_000_000), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		"echo '[download] Destination: " + target + "'\n" +
		"echo '[download] 100.0% of 5.00MiB at 1.00MiB/s ETA 00:00'\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "yt-dlp"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	e := NewEngine(1, 1<<30)
	j := &Job{URL: "https://youtube.com/watch?v=x", Dir: out, Kind: KindMedia}
	if err := e.runYtdlp(context.Background(), j); err != nil {
		t.Fatalf("job failed: %v", err)
	}
	if j.Total != 9_000_000 {
		t.Errorf("Total = %d, want the merged file's 9000000 bytes", j.Total)
	}
}
