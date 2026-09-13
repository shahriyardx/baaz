package downloader

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSplitSegments(t *testing.T) {
	segs := splitSegments(100, 8, 10)
	if len(segs) != 8 {
		t.Fatalf("want 8 segments, got %d", len(segs))
	}
	var covered int64
	prevEnd := int64(-1)
	for _, s := range segs {
		if s.Start != prevEnd+1 {
			t.Fatalf("gap: segment starts at %d, previous ended at %d", s.Start, prevEnd)
		}
		covered += s.End - s.Start + 1
		prevEnd = s.End
	}
	if covered != 100 || prevEnd != 99 {
		t.Fatalf("coverage %d, last end %d", covered, prevEnd)
	}

	// small file: capped by minSplit
	segs = splitSegments(100, 8, 60)
	if len(segs) != 1 {
		t.Fatalf("want 1 segment for tiny file, got %d", len(segs))
	}
}

func testPayload(t *testing.T, size int) []byte {
	t.Helper()
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	return data
}

func runJob(t *testing.T, url string, minSplit int64) (*Job, []byte) {
	t.Helper()
	dir := t.TempDir()
	j := &Job{ID: "test1234", URL: url, Dir: dir, State: StateQueued, Total: -1, CreatedAt: time.Now()}
	e := NewEngine(8, minSplit)
	if err := e.Run(context.Background(), j); err != nil {
		t.Fatalf("run: %v", err)
	}
	if j.GetState() != StateDone {
		t.Fatalf("state = %s, want done", j.GetState())
	}
	got, err := os.ReadFile(j.FinalPath)
	if err != nil {
		t.Fatal(err)
	}
	return j, got
}

func TestSegmentedDownload(t *testing.T) {
	data := testPayload(t, 1<<20+12345)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.ServeContent implements Range
		http.ServeContent(w, r, "file.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()

	_, got := runJob(t, srv.URL+"/file.bin", 64<<10)
	if sha256.Sum256(got) != sha256.Sum256(data) {
		t.Fatal("checksum mismatch")
	}
}

func TestNoRangeFallback(t *testing.T) {
	data := testPayload(t, 300<<10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="plain.bin"`)
		w.WriteHeader(http.StatusOK) // ignores Range entirely
		w.Write(data)
	}))
	defer srv.Close()

	j, got := runJob(t, srv.URL+"/x", 64<<10)
	if !bytes.Equal(got, data) {
		t.Fatal("content mismatch")
	}
	if filepath.Base(j.FinalPath) != "plain.bin" {
		t.Fatalf("filename = %s, want plain.bin (from Content-Disposition)", filepath.Base(j.FinalPath))
	}
}

// Server claims ranges on the probe but returns 200 on segment requests:
// engine must fall back to a single stream and still produce a correct file.
func TestLyingRangeServer(t *testing.T) {
	data := testPayload(t, 500<<10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "bytes=0-0" {
			http.ServeContent(w, r, "f.bin", time.Time{}, bytes.NewReader(data))
			return
		}
		w.Write(data)
	}))
	defer srv.Close()

	_, got := runJob(t, srv.URL+"/f.bin", 64<<10)
	if !bytes.Equal(got, data) {
		t.Fatal("content mismatch after fallback")
	}
}

func TestPauseResume(t *testing.T) {
	data := testPayload(t, 2<<20)
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// throttle so the pause lands mid-transfer
		http.ServeContent(w, r, "f.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer slow.Close()

	dir := t.TempDir()
	j := &Job{ID: "pr1", URL: slow.URL + "/f.bin", Dir: dir, State: StateQueued, Total: -1, CreatedAt: time.Now()}
	e := NewEngine(4, 64<<10)

	done := make(chan error, 1)
	go func() { done <- e.Run(context.Background(), j) }()
	time.Sleep(50 * time.Millisecond)
	j.Stop()
	<-done
	if s := j.GetState(); s != StatePaused && s != StateDone {
		t.Fatalf("state after stop = %s", s)
	}
	if j.GetState() == StatePaused {
		// resume: same job, same segments
		if err := e.Run(context.Background(), j); err != nil {
			t.Fatalf("resume: %v", err)
		}
	}
	got, err := os.ReadFile(j.FinalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("checksum mismatch after resume")
	}
}

func TestNameCollision(t *testing.T) {
	data := testPayload(t, 10 << 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "dup.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer srv.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "dup.bin"), []byte("old"), 0o644)
	j := &Job{ID: "c1", URL: srv.URL + "/dup.bin", Dir: dir, State: StateQueued, Total: -1, CreatedAt: time.Now()}
	if err := NewEngine(2, 1<<10).Run(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(j.FinalPath) != "dup (1).bin" {
		t.Fatalf("collision name = %s, want dup (1).bin", filepath.Base(j.FinalPath))
	}
}
