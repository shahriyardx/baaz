package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	maxAttempts = 5
	copyBufSize = 128 << 10
)

// downloadSegment fetches one byte range with retries, resuming from the
// segment's Written offset on each attempt. Progress resets the attempt
// counter so a slow-but-alive connection is never given up on.
func (e *Engine) downloadSegment(ctx context.Context, j *Job, f *os.File, seg *Segment) error {
	attempts := 0
	backoff := time.Second
	for {
		before := seg.written()
		err := e.trySegment(ctx, j, f, seg)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, errRangeNotSupported) {
			return err
		}
		if seg.written() > before {
			attempts = 0
			backoff = time.Second
		}
		attempts++
		if attempts >= maxAttempts {
			return fmt.Errorf("segment %d-%d: %w", seg.Start, seg.End, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

func (e *Engine) trySegment(ctx context.Context, j *Job, f *os.File, seg *Segment) error {
	if seg.complete() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.URL, nil)
	if err != nil {
		return err
	}
	applyHeaders(req, j.Headers)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", seg.Start+seg.written(), seg.End))
	resp, err := e.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusPartialContent:
	case http.StatusOK:
		return errRangeNotSupported
	default:
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	if err := copyToSegment(ctx, j, f, seg, resp.Body); err != nil {
		return err
	}
	if !seg.complete() {
		return io.ErrUnexpectedEOF
	}
	return nil
}

// copyToSegment streams r into f at the segment's current offset, updating
// Written atomically so progress reporting sees live counts. While the job
// is soft-paused it stops reading — TCP backpressure holds the transfer
// without closing the connection.
func copyToSegment(ctx context.Context, j *Job, f *os.File, seg *Segment, r io.Reader) error {
	buf := make([]byte, copyBufSize)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if j.SoftPaused() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(250 * time.Millisecond):
			}
			continue
		}
		n, rerr := r.Read(buf)
		if n > 0 {
			off := seg.Start + seg.written()
			if _, werr := f.WriteAt(buf[:n], off); werr != nil {
				return werr
			}
			seg.addWritten(int64(n))
			if seg.End >= 0 && seg.written() > seg.End-seg.Start+1 {
				return fmt.Errorf("server sent more bytes than requested")
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}
