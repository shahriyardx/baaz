// Package ipc defines the JSON-lines protocol spoken over the daemon's unix
// socket. protocol.go is the single source of truth for the schema consumed
// by the CLI, `baaz watch`, the native-messaging host, and the bar widget.
package ipc

type Request struct {
	Cmd      string            `json:"cmd"` // add|list|pause|resume|cancel|status|watch|ping|getconfig|setconfig
	URL      string            `json:"url,omitempty"`
	Filename string            `json:"filename,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	ID       string            `json:"id,omitempty"`
	Size     int64             `json:"size,omitempty"`     // advertised size, for min-size policy
	Format   string            `json:"format,omitempty"`   // media quality preset: best|1080|720|480|audio
	Settings map[string]string `json:"settings,omitempty"` // setconfig: key -> value
}

// Settings is the daemon-owned configuration exposed to every UI.
type Settings struct {
	Intercept   bool   `json:"intercept"`
	Segments    int    `json:"segments"`
	MaxActive   int    `json:"maxActive"`
	MinSizeMB   int    `json:"minSizeMB"`
	DownloadDir string `json:"downloadDir"`
	Categorize  bool   `json:"categorize"`
}

// ErrRejected marks policy rejections (intercept off, below min size) so the
// extension can fall back to a plain browser download silently.
const ErrRejected = "rejected: "

type Response struct {
	OK       bool      `json:"ok"`
	Error    string    `json:"error,omitempty"`
	ID       string    `json:"id,omitempty"`
	Snapshot *Snapshot `json:"snapshot,omitempty"`
}

// SegmentInfo is one byte range of a job. Sent only while a job is in
// flight — finished jobs drop their segments — so a UI can show the file
// genuinely being pulled in parallel pieces.
type SegmentInfo struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"` // -1 when the length is unknown
}

type JobInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	Total int64  `json:"total"`
	Done  int64  `json:"done"`
	Speed int64  `json:"speed"` // bytes/sec, 3s rolling average
	ETA   int64  `json:"eta"`   // seconds, -1 when unknown
	Dir   string `json:"dir"`
	Error string `json:"error,omitempty"`

	Segments []SegmentInfo `json:"segments,omitempty"`
}

// Snapshot is the full state pushed to `watch` subscribers and returned by
// `baaz status --json`.
type Snapshot struct {
	Type       string    `json:"type"` // always "snapshot"
	Active     int       `json:"active"`
	TotalSpeed int64     `json:"totalSpeed"`
	Jobs       []JobInfo `json:"jobs"`   // queued/active/paused/failed
	Recent     []JobInfo `json:"recent"` // last 10 completed
	Settings   Settings  `json:"settings"`
}
