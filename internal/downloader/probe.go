package downloader

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type probeResult struct {
	Total    int64 // -1 when unknown
	Ranged   bool
	Filename string        // from Content-Disposition or URL; may be ""
	Body     io.ReadCloser // non-nil only when the response is reusable as a full-body stream
}

// probe issues GET with "Range: bytes=0-0". A 206 proves range support and
// carries the total size in Content-Range. A 200 means the server ignored the
// header — its body IS the whole file, so it is handed back for reuse.
func probe(ctx context.Context, client *http.Client, rawURL string, headers map[string]string) (*probeResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	applyHeaders(req, headers)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	pr := &probeResult{Total: -1, Filename: filenameFrom(resp, rawURL)}
	switch resp.StatusCode {
	case http.StatusPartialContent:
		defer resp.Body.Close()
		io.Copy(io.Discard, io.LimitReader(resp.Body, 2))
		pr.Ranged = true
		pr.Total = parseContentRangeTotal(resp.Header.Get("Content-Range"))
		if pr.Total <= 0 {
			pr.Ranged = false // can't split without a size
		}
		return pr, nil
	case http.StatusOK:
		pr.Body = resp.Body
		if resp.ContentLength > 0 {
			pr.Total = resp.ContentLength
		}
		return pr, nil
	default:
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
}

// parseContentRangeStart extracts S from "bytes S-E/…".
func parseContentRangeStart(v string) (int64, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "bytes ")
	i := strings.IndexByte(v, '-')
	if i < 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v[:i]), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseContentRangeTotal extracts N from "bytes 0-0/N"; -1 on failure or "*".
func parseContentRangeTotal(v string) int64 {
	i := strings.LastIndexByte(v, '/')
	if i < 0 {
		return -1
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v[i+1:]), 10, 64)
	if err != nil {
		return -1
	}
	return n
}

func filenameFrom(resp *http.Response, rawURL string) string {
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fn := sanitizeName(params["filename"]); fn != "" {
				return fn
			}
		}
	}
	if u, err := url.Parse(rawURL); err == nil {
		if fn := sanitizeName(path.Base(u.Path)); fn != "" && fn != "/" && fn != "." {
			return fn
		}
	}
	return ""
}

// sanitizeName strips any path components so a hostile header can't escape
// the download directory.
func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if name == "/" || name == "." || name == ".." {
		return ""
	}
	return name
}
