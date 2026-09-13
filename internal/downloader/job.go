package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type State string

const (
	StateQueued   State = "queued"
	StateActive   State = "active"
	StatePaused   State = "paused"
	StateDone     State = "done"
	StateFailed   State = "failed"
	StateCanceled State = "canceled"
)

type Segment struct {
	Start   int64 `json:"start"`
	End     int64 `json:"end"` // inclusive; -1 = unknown length (single stream)
	Written int64 `json:"written"`
}

func (s *Segment) written() int64     { return atomic.LoadInt64(&s.Written) }
func (s *Segment) addWritten(n int64) { atomic.AddInt64(&s.Written, n) }

func (s *Segment) complete() bool {
	return s.End >= 0 && s.written() >= s.End-s.Start+1
}

type Job struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Filename    string            `json:"filename"` // final basename; may be refined by probe
	Dir         string            `json:"dir"`      // destination directory
	Headers     map[string]string `json:"headers"`
	State       State             `json:"state"`
	Total       int64             `json:"total"` // -1 = unknown
	Segments    []*Segment        `json:"segments,omitempty"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	CompletedAt *time.Time        `json:"completedAt,omitempty"`
	FinalPath   string            `json:"finalPath,omitempty"`
	NoRange     bool              `json:"noRange,omitempty"` // single-stream job (no usable Range support)
	Kind        string            `json:"kind,omitempty"`    // "" = http, "media" = yt-dlp
	Format      string            `json:"format,omitempty"`  // media preset: best|1080|720|480|audio
	Categorize  bool              `json:"categorize,omitempty"` // sort into <dir>/baaz/<Category>

	mu        sync.Mutex
	cancel    context.CancelFunc
	softPause atomic.Bool // pause by not reading; connection stays open
}

// SetSoftPause suspends/continues a running single-stream transfer without
// dropping the connection — the only lossless pause when ranges don't work.
func (j *Job) SetSoftPause(v bool) { j.softPause.Store(v) }
func (j *Job) SoftPaused() bool    { return j.softPause.Load() }

// errPausedConnLost: the server dropped a soft-paused connection; the job
// should land in paused (resume retries with a Range request), not failed.
var errPausedConnLost = errors.New("connection lost while paused")

func (j *Job) Done() int64 {
	var n int64
	j.mu.Lock()
	segs := j.Segments
	j.mu.Unlock()
	for _, s := range segs {
		n += s.written()
	}
	return n
}

func (j *Job) GetState() State {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.State
}

func (j *Job) SetState(s State) {
	j.mu.Lock()
	j.State = s
	j.mu.Unlock()
}

// Cancel stops the running transfer, if any.
func (j *Job) Stop() {
	j.mu.Lock()
	c := j.cancel
	j.mu.Unlock()
	if c != nil {
		c()
	}
}

// In-flight files live in a hidden dir inside the download root: invisible
// in the file manager, yet on the same filesystem so the final rename is
// instant.
func (j *Job) tmpDir() string { return filepath.Join(j.Dir, ".baaz-tmp") }

func (j *Job) partPath() string {
	return filepath.Join(j.tmpDir(), fmt.Sprintf("%s.part", j.ID))
}

var errRangeNotSupported = errors.New("server does not honor Range requests")

// Engine executes jobs. Safe for concurrent Run calls on distinct jobs.
type Engine struct {
	Client       *http.Client
	Segments     int
	MinSplitSize int64
	// OnProgress, if set, is invoked roughly once per second from the running
	// job so the caller can persist state.
	OnProgress func(*Job)
}

func NewEngine(segments int, minSplit int64) *Engine {
	return &Engine{
		Client: &http.Client{
			// No overall timeout: downloads are long-lived. Dial/TLS timeouts
			// come from DefaultTransport.
			Transport: http.DefaultTransport,
		},
		Segments:     segments,
		MinSplitSize: minSplit,
	}
}

// Run performs the download for job j until done, failed, or canceled via
// j.Stop(). It updates j.State and returns the terminal error, if any.
// The caller owns persistence; Run only mutates the in-memory job.
func (e *Engine) Run(parent context.Context, j *Job) error {
	ctx, cancel := context.WithCancel(parent)
	j.mu.Lock()
	j.cancel = cancel
	j.State = StateActive
	j.mu.Unlock()
	defer cancel()

	err := e.run(ctx, j)

	j.mu.Lock()
	defer j.mu.Unlock()
	j.cancel = nil
	switch {
	case err == nil:
		now := time.Now()
		j.State = StateDone
		j.CompletedAt = &now
		j.Error = ""
		if j.Total <= 0 { // unknown-length stream: the bytes on disk are the size
			var n int64
			for _, s := range j.Segments {
				n += s.written()
			}
			j.Total = n
		}
		j.Segments = nil // drop segment detail; keep the record for "recent"
	case errors.Is(err, errPausedConnLost) && j.State == StateActive:
		j.State = StatePaused
		j.softPause.Store(false)
	case ctx.Err() != nil && j.State == StateActive:
		// stopped via Stop() — pause/cancel decided by the caller
		j.State = StatePaused
	case j.State == StateActive:
		j.State = StateFailed
		j.Error = err.Error()
	}
	return err
}

func (e *Engine) run(ctx context.Context, j *Job) error {
	if j.Kind == KindMedia {
		return e.runYtdlp(ctx, j) // yt-dlp writes the final file itself
	}
	pr, err := probe(ctx, e.Client, j.URL, j.Headers)
	if err != nil {
		return err
	}
	if j.Filename == "" {
		j.Filename = pr.Filename
	}
	j.mu.Lock()
	j.Total = pr.Total
	resuming := len(j.Segments) > 0 && pr.Ranged && pr.Total > 0
	j.mu.Unlock()

	if err := os.MkdirAll(j.tmpDir(), 0o755); err != nil {
		if pr.Body != nil {
			pr.Body.Close()
		}
		return err
	}

	if pr.Ranged && pr.Total > 0 {
		if pr.Body != nil {
			pr.Body.Close() // segmented path re-requests; discard probe body
		}
		if !resuming {
			j.mu.Lock()
			j.Segments = splitSegments(pr.Total, e.Segments, e.MinSplitSize)
			j.mu.Unlock()
		}
		err = e.runSegmented(ctx, j)
		if errors.Is(err, errRangeNotSupported) {
			// Server lied on a segment request: restart as one stream.
			j.mu.Lock()
			j.NoRange = true
			j.Segments = []*Segment{{Start: 0, End: pr.Total - 1}}
			j.mu.Unlock()
			err = e.runSingle(ctx, j, nil)
		}
	} else {
		j.mu.Lock()
		end := int64(-1)
		if pr.Total > 0 {
			end = pr.Total - 1
		}
		j.NoRange = true
		if len(j.Segments) == 0 {
			j.Segments = []*Segment{{Start: 0, End: end}}
		}
		j.Segments[0].End = end
		j.mu.Unlock()
		err = e.runSingle(ctx, j, pr.Body)
	}
	if err != nil {
		return err
	}
	return j.finish()
}

func (e *Engine) runSegmented(ctx context.Context, j *Job) error {
	f, err := os.OpenFile(j.partPath(), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Truncate(j.Total); err != nil {
		return err
	}

	gctx, gcancel := context.WithCancel(ctx)
	defer gcancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(j.Segments))
	for _, seg := range j.Segments {
		if seg.complete() {
			continue
		}
		wg.Add(1)
		go func(s *Segment) {
			defer wg.Done()
			if err := e.downloadSegment(gctx, j, f, s); err != nil {
				errCh <- err
				gcancel() // one segment giving up stops the rest
			}
		}(seg)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-done:
			select {
			case err := <-errCh:
				return err
			default:
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return f.Sync()
		case <-tick.C:
			if e.OnProgress != nil {
				e.OnProgress(j)
			}
		}
	}
}

func (e *Engine) runSingle(ctx context.Context, j *Job, probeBody io.ReadCloser) error {
	seg := j.Segments[0]

	// Pick the byte source. With bytes already on disk, first ask the server
	// to continue from that offset — some honor Range even when the total
	// size is unknown. A 200 means start over.
	var body io.ReadCloser
	truncate := true
	switch {
	case probeBody != nil:
		seg.Written = 0
		body = probeBody
	case seg.written() > 0:
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(req, j.Headers)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", seg.written()))
		resp, err := e.Client.Do(req)
		if err != nil {
			return err
		}
		switch resp.StatusCode {
		case http.StatusPartialContent:
			if start, ok := parseContentRangeStart(resp.Header.Get("Content-Range")); ok && start == seg.written() {
				body = resp.Body // append from where we stopped
				truncate = false
			} else {
				resp.Body.Close()
				return fmt.Errorf("server resumed at wrong offset")
			}
		case http.StatusOK:
			seg.Written = 0 // stream restarts; so do we
			body = resp.Body
		default:
			resp.Body.Close()
			return fmt.Errorf("HTTP %s", resp.Status)
		}
	default:
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(req, j.Headers)
		resp, err := e.Client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			return fmt.Errorf("HTTP %s", resp.Status)
		}
		body = resp.Body
	}
	defer body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	if truncate {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(j.partPath(), flags, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				if e.OnProgress != nil {
					e.OnProgress(j)
				}
			}
		}
	}()

	if err := copyToSegment(ctx, j, f, seg, body); err != nil {
		// A soft-paused stream that the server hung up on becomes a paused
		// job; resume goes through the Range-append path above.
		if j.SoftPaused() && ctx.Err() == nil {
			return errPausedConnLost
		}
		return err
	}
	return f.Sync()
}

// finish renames the .part file to the final name, avoiding collisions.
func (j *Job) finish() error {
	name := j.Filename
	if name == "" {
		name = "download.bin"
	}
	destDir := j.Dir
	if j.Categorize {
		destDir = CategorizedDir(j.Dir, name)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
	}
	dest := filepath.Join(destDir, name)
	ext := filepath.Ext(name)
	base := name[:len(name)-len(ext)]
	for i := 1; ; i++ {
		if _, err := os.Lstat(dest); os.IsNotExist(err) {
			break
		}
		dest = filepath.Join(destDir, fmt.Sprintf("%s (%d)%s", base, i, ext))
	}
	if err := os.Rename(j.partPath(), dest); err != nil {
		return err
	}
	j.mu.Lock()
	j.FinalPath = dest
	j.Filename = filepath.Base(dest)
	j.mu.Unlock()
	return nil
}

// CleanupPart removes the partial file (cancel path).
func (j *Job) CleanupPart() { os.Remove(j.partPath()) }

func splitSegments(total int64, n int, minSplit int64) []*Segment {
	if minSplit < 1 {
		minSplit = 1
	}
	maxN := total / minSplit
	if maxN < 1 {
		maxN = 1
	}
	if int64(n) > maxN {
		n = int(maxN)
	}
	if n < 1 {
		n = 1
	}
	segs := make([]*Segment, 0, n)
	base := total / int64(n)
	rem := total % int64(n)
	var off int64
	for i := 0; i < n; i++ {
		size := base
		if int64(i) < rem {
			size++
		}
		segs = append(segs, &Segment{Start: off, End: off + size - 1})
		off += size
	}
	return segs
}

func applyHeaders(req *http.Request, h map[string]string) {
	for k, v := range h {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
}
