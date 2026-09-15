// Package daemon manages the download queue: scheduling, persistence,
// speed accounting, and snapshot fan-out to watch subscribers.
package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"baaz/internal/config"
	"baaz/internal/downloader"
	"baaz/internal/ipc"
)

const (
	speedWindow   = 3 * time.Second
	tickInterval  = 500 * time.Millisecond
	saveInterval  = time.Second
	recentCount   = 10
	doneKeepCount = 50
)

type speedSample struct {
	t    time.Time
	done int64
}

type Manager struct {
	cfg *config.Config
	eng *downloader.Engine

	mu           sync.Mutex
	jobs         map[string]*downloader.Job
	order        []string // creation order
	active       int
	cancelIntent map[string]bool
	samples      map[string][]speedSample
	lastSave     map[string]time.Time
	subs         map[chan *ipc.Snapshot]struct{}
	// setup is non-empty while the one-time video tools are being fetched.
	setup string
}

func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		cfg:          cfg,
		jobs:         map[string]*downloader.Job{},
		cancelIntent: map[string]bool{},
		samples:      map[string][]speedSample{},
		lastSave:     map[string]time.Time{},
		subs:         map[chan *ipc.Snapshot]struct{}{},
	}
	m.eng = downloader.NewEngine(cfg.Segments, cfg.MinSplitSize)
	m.eng.Limiter.SetRate(int64(cfg.SpeedLimitKB) << 10)
	m.eng.ToolsDir = filepath.Join(config.DataDir(), "bin")
	m.eng.OnProgress = func(j *downloader.Job) { m.persistThrottled(j) }

	// Crash recovery: anything found mid-flight becomes paused; user resumes.
	for _, j := range downloader.LoadAll(config.JobsDir()) {
		if s := j.GetState(); s == downloader.StateActive || s == downloader.StateQueued {
			j.SetState(downloader.StatePaused)
			downloader.Save(config.JobsDir(), j)
		}
		m.jobs[j.ID] = j
		m.order = append(m.order, j.ID)
	}
	sort.SliceStable(m.order, func(a, b int) bool {
		return m.jobs[m.order[a]].CreatedAt.Before(m.jobs[m.order[b]].CreatedAt)
	})
	return m
}

// Start launches the scheduler tick loop.
// prepareMediaTools fetches yt-dlp and ffmpeg in the background, once, when
// the app starts. Doing it here rather than at the first video download
// means the wait happens while nobody is watching, instead of in front of
// someone who has just clicked Download.
//
// It never blocks anything: a download started meanwhile waits on the same
// lock and carries on, and a failure is left alone — the download path will
// try again, and say so there, which is where it matters.
func (m *Manager) prepareMediaTools(ctx context.Context) {
	go func() {
		err := m.eng.EnsureMediaTools(ctx, func(s string) {
			m.mu.Lock()
			m.setup = s
			m.mu.Unlock()
			m.broadcast()
		})
		m.mu.Lock()
		m.setup = ""
		m.mu.Unlock()
		if err != nil {
			log.Printf("could not set up the video tools: %v", err)
		}
		m.broadcast()
	}()
}

func (m *Manager) Start(ctx context.Context) {
	m.prepareMediaTools(ctx)
	go func() {
		tick := time.NewTicker(tickInterval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				m.tick()
			}
		}
	}()
}

// Shutdown stops active transfers so their offsets persist as paused.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	var running []*downloader.Job
	for _, j := range m.jobs {
		if j.GetState() == downloader.StateActive {
			running = append(running, j)
		}
	}
	m.mu.Unlock()
	for _, j := range running {
		j.Stop()
	}
	// Give Run() a moment to flush state.
	time.Sleep(300 * time.Millisecond)
	for _, j := range running {
		downloader.Save(config.JobsDir(), j)
	}
}

func newID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// AddBrowser is the extension's entry point: it enforces intercept and
// min-size policy so a rejection falls back to a plain browser download.
func (m *Manager) AddBrowser(url, filename string, headers map[string]string, size int64) (string, error) {
	m.mu.Lock()
	on := m.cfg.InterceptOn()
	minBytes := int64(m.cfg.MinSizeMB) << 20
	m.mu.Unlock()
	if !on {
		return "", fmt.Errorf("%sintercept is off", ipc.ErrRejected)
	}
	if size > 0 && size < minBytes {
		return "", fmt.Errorf("%sbelow min size", ipc.ErrRejected)
	}
	return m.Add(url, filename, headers, "")
}

// checkURL rejects what cannot possibly be downloaded, before a job exists.
// Queuing it anyway meant the list filled with entries that failed a moment
// later carrying Go's own wording — `unsupported protocol scheme ""` is not
// something to show anyone who mistyped an address.
func checkURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("no link given")
	}
	u, err := neturl.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("that does not look like a link")
	}
	switch u.Scheme {
	case "http", "https":
	case "":
		return "", fmt.Errorf("that does not look like a link — it needs to start with http:// or https://")
	default:
		return "", fmt.Errorf("%s links are not supported, only http and https", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("that link has no website in it")
	}
	return raw, nil
}

func (m *Manager) Add(rawURL, filename string, headers map[string]string, format string) (string, error) {
	url, err := checkURL(rawURL)
	if err != nil {
		return "", err
	}
	j := &downloader.Job{
		ID:        newID(),
		URL:       url,
		Filename:  filename,
		Dir:       m.cfg.ResolvedDownloadDir(),
		Headers:   headers,
		State:     downloader.StateQueued,
		Total:     -1,
		CreatedAt: time.Now(),
	}
	if downloader.IsMediaURL(url) {
		j.Kind = downloader.KindMedia
		j.Format = format
	}
	m.mu.Lock()
	j.Categorize = m.cfg.CategorizeOn()
	m.mu.Unlock()
	m.mu.Lock()
	m.jobs[j.ID] = j
	m.order = append(m.order, j.ID)
	m.mu.Unlock()
	downloader.Save(config.JobsDir(), j)
	m.schedule()
	m.broadcast()
	name := filename
	if name == "" {
		name = url
	}
	notify("Download started", name)
	return j.ID, nil
}

func (m *Manager) Pause(id string) error {
	j, err := m.get(id)
	if err != nil {
		return err
	}
	switch j.GetState() {
	case downloader.StateActive:
		if j.NoRange {
			// No ranges = a dropped connection can't continue, so pause by
			// holding the connection open and reading nothing.
			j.SetSoftPause(true)
			m.broadcast()
			return nil
		}
		j.Stop() // Run() flips it to paused and runJob persists
	case downloader.StateQueued:
		j.SetState(downloader.StatePaused)
		downloader.Save(config.JobsDir(), j)
		m.broadcast()
	default:
		return fmt.Errorf("job %s is %s", id, j.GetState())
	}
	return nil
}

func (m *Manager) Resume(id string) error {
	j, err := m.get(id)
	if err != nil {
		return err
	}
	if j.SoftPaused() {
		j.SetSoftPause(false)
		m.broadcast()
		return nil
	}
	s := j.GetState()
	if s != downloader.StatePaused && s != downloader.StateFailed {
		return fmt.Errorf("job %s is %s", id, s)
	}
	j.SetState(downloader.StateQueued)
	downloader.Save(config.JobsDir(), j)
	m.schedule()
	m.broadcast()
	return nil
}

func (m *Manager) Cancel(id string) error {
	j, err := m.get(id)
	if err != nil {
		return err
	}
	if j.GetState() == downloader.StateActive {
		m.mu.Lock()
		m.cancelIntent[id] = true
		m.mu.Unlock()
		j.Stop() // runJob finishes the cleanup
		return nil
	}
	m.remove(j)
	m.broadcast()
	return nil
}

// Clear drops every completed job record; downloaded files stay on disk.
func (m *Manager) Clear() error {
	m.mu.Lock()
	var done []*downloader.Job
	for _, j := range m.jobs {
		if j.GetState() == downloader.StateDone {
			done = append(done, j)
		}
	}
	m.mu.Unlock()
	for _, j := range done {
		m.forget(j)
	}
	m.broadcast()
	return nil
}

// Delete removes one finished/failed job AND its file from the filesystem.
func (m *Manager) Delete(id string) error {
	j, err := m.get(id)
	if err != nil {
		return err
	}
	s := j.GetState()
	if s != downloader.StateDone && s != downloader.StateFailed {
		return fmt.Errorf("job %s is %s — cancel it instead", id, s)
	}
	if s == downloader.StateDone && j.FinalPath != "" {
		if err := os.Remove(j.FinalPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	j.CleanupPart()
	m.forget(j)
	m.broadcast()
	return nil
}

// forget removes a job's record and state file (never the downloaded file).
func (m *Manager) forget(j *downloader.Job) {
	downloader.Remove(config.JobsDir(), j)
	m.mu.Lock()
	delete(m.jobs, j.ID)
	delete(m.samples, j.ID)
	for i, id := range m.order {
		if id == j.ID {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	m.mu.Unlock()
}

func (m *Manager) remove(j *downloader.Job) {
	j.SetState(downloader.StateCanceled)
	j.CleanupPart()
	downloader.Remove(config.JobsDir(), j)
	m.mu.Lock()
	delete(m.jobs, j.ID)
	delete(m.samples, j.ID)
	for i, id := range m.order {
		if id == j.ID {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	m.mu.Unlock()
}

func (m *Manager) get(id string) (*downloader.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("no such job: %s", id)
	}
	return j, nil
}

// schedule starts queued jobs while slots are free.
func (m *Manager) schedule() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		if m.active >= m.cfg.MaxActive {
			return
		}
		j := m.jobs[id]
		if j.GetState() != downloader.StateQueued {
			continue
		}
		j.SetState(downloader.StateActive)
		m.active++
		go m.runJob(j)
	}
}

func (m *Manager) runJob(j *downloader.Job) {
	m.eng.Run(context.Background(), j)

	m.mu.Lock()
	m.active--
	canceled := m.cancelIntent[j.ID]
	delete(m.cancelIntent, j.ID)
	delete(m.samples, j.ID)
	m.mu.Unlock()

	if canceled {
		m.remove(j)
	} else {
		downloader.Save(config.JobsDir(), j)
		switch j.GetState() {
		case downloader.StateDone:
			notify("Download finished", j.Filename)
			m.pruneDone()
		case downloader.StateFailed:
			notify("Download failed", j.Filename+": "+j.Error)
		}
	}
	m.schedule()
	m.broadcast()
}

// pruneDone caps stored completed-job records.
func (m *Manager) pruneDone() {
	m.mu.Lock()
	var done []*downloader.Job
	for _, j := range m.jobs {
		if j.GetState() == downloader.StateDone {
			done = append(done, j)
		}
	}
	sort.Slice(done, func(a, b int) bool {
		return completedAt(done[a]).After(completedAt(done[b]))
	})
	var evict []*downloader.Job
	if len(done) > doneKeepCount {
		evict = done[doneKeepCount:]
	}
	for _, j := range evict {
		delete(m.jobs, j.ID)
		for i, id := range m.order {
			if id == j.ID {
				m.order = append(m.order[:i], m.order[i+1:]...)
				break
			}
		}
	}
	m.mu.Unlock()
	for _, j := range evict {
		downloader.Remove(config.JobsDir(), j)
	}
}

func completedAt(j *downloader.Job) time.Time {
	if j.CompletedAt != nil {
		return *j.CompletedAt
	}
	return j.CreatedAt
}

func (m *Manager) tick() {
	now := time.Now()
	m.mu.Lock()
	activeJobs := 0
	for _, id := range m.order {
		j := m.jobs[id]
		if j.GetState() != downloader.StateActive {
			continue
		}
		activeJobs++
		s := append(m.samples[id], speedSample{now, j.Done()})
		for len(s) > 1 && now.Sub(s[0].t) > speedWindow {
			s = s[1:]
		}
		m.samples[id] = s
	}
	m.mu.Unlock()
	if activeJobs > 0 {
		m.broadcast()
	}
}

func (m *Manager) persistThrottled(j *downloader.Job) {
	m.mu.Lock()
	last := m.lastSave[j.ID]
	now := time.Now()
	if now.Sub(last) < saveInterval {
		m.mu.Unlock()
		return
	}
	m.lastSave[j.ID] = now
	m.mu.Unlock()
	downloader.Save(config.JobsDir(), j)
}

func (m *Manager) speedOf(id string) int64 {
	s := m.samples[id]
	if len(s) < 2 {
		return 0
	}
	dt := s[len(s)-1].t.Sub(s[0].t).Seconds()
	if dt <= 0 {
		return 0
	}
	db := s[len(s)-1].done - s[0].done
	if db < 0 {
		return 0
	}
	return int64(float64(db) / dt)
}

// Snapshot builds the full state view shared by status and watch.
// Formats reports the resolutions a media URL offers, so a UI can show the
// ones that exist rather than a fixed list.
func (m *Manager) Formats(rawURL string) ([]ipc.QualityInfo, error) {
	url, err := checkURL(rawURL)
	if err != nil {
		return nil, err
	}
	// Extraction is a network round trip; the caller is a menu waiting to
	// open, so it must not hang on a slow or wrong link.
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	qualities, err := m.eng.AvailableQualities(ctx, url)
	if err != nil {
		return nil, err
	}
	out := make([]ipc.QualityInfo, 0, len(qualities))
	for _, q := range qualities {
		out = append(out, ipc.QualityInfo{Label: q.Label, Codec: q.Codec})
	}
	return out, nil
}

func (m *Manager) Snapshot() *ipc.Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := &ipc.Snapshot{
		Type: "snapshot", Jobs: []ipc.JobInfo{}, Recent: []ipc.JobInfo{},
		Setup: m.setup,
		Settings: ipc.Settings{
			Intercept:    m.cfg.InterceptOn(),
			Segments:     m.cfg.Segments,
			MaxActive:    m.cfg.MaxActive,
			MinSizeMB:    m.cfg.MinSizeMB,
			DownloadDir:  m.cfg.DownloadDir,
			Categorize:   m.cfg.CategorizeOn(),
			SpeedLimitKB: m.cfg.SpeedLimitKB,
		},
	}
	var recent []*downloader.Job
	for _, id := range m.order {
		j := m.jobs[id]
		state := j.GetState()
		if state == downloader.StateDone {
			recent = append(recent, j)
			continue
		}
		info := m.jobInfo(j, state)
		if state == downloader.StateActive {
			snap.Active++
			snap.TotalSpeed += info.Speed
		}
		snap.Jobs = append(snap.Jobs, info)
	}
	sort.Slice(recent, func(a, b int) bool {
		return completedAt(recent[a]).After(completedAt(recent[b]))
	})
	if len(recent) > recentCount {
		recent = recent[:recentCount]
	}
	for _, j := range recent {
		snap.Recent = append(snap.Recent, m.jobInfo(j, downloader.StateDone))
	}
	return snap
}

func (m *Manager) jobInfo(j *downloader.Job, state downloader.State) ipc.JobInfo {
	if state == downloader.StateActive && j.SoftPaused() {
		state = downloader.StatePaused // held connection reads as paused
	}
	info := ipc.JobInfo{
		ID:    j.ID,
		Name:  j.Filename,
		State: string(state),
		Total: j.Total,
		Done:  j.Done(),
		Speed: m.speedOf(j.ID),
		ETA:   -1,
		Dir:   j.Dir,
		Error: j.Error,
	}
	if info.Name == "" {
		info.Name = j.URL
	}
	info.Note = j.Note()
	info.URL = j.URL
	info.Kind = j.Kind
	info.NoRange = j.NoRange
	info.CreatedAt = j.CreatedAt.Format(time.RFC3339)
	if j.CompletedAt != nil {
		info.CompletedAt = j.CompletedAt.Format(time.RFC3339)
	}
	if j.FinalPath != "" {
		info.Dir = filepath.Dir(j.FinalPath)
	}
	if state == downloader.StateDone {
		if j.Total > 0 {
			info.Done = j.Total
		} else {
			info.Total = info.Done // unknown-length stream: final size is the truth
		}
	}
	if info.Speed > 0 && info.Total > 0 && info.Done <= info.Total {
		info.ETA = (info.Total - info.Done) / info.Speed
	}
	// Only for jobs still in flight: a finished job has already dropped its
	// segments, and shipping them for every recent entry would bloat a
	// snapshot that goes out twice a second.
	if state != downloader.StateDone {
		for _, sp := range j.SegmentProgress() {
			info.Segments = append(info.Segments, ipc.SegmentInfo{Done: sp.Done, Total: sp.Total})
		}
	}
	return info
}

// Settings returns the live configuration.
func (m *Manager) Settings() ipc.Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return ipc.Settings{
		Intercept:   m.cfg.InterceptOn(),
		Segments:    m.cfg.Segments,
		MaxActive:   m.cfg.MaxActive,
		MinSizeMB:   m.cfg.MinSizeMB,
		DownloadDir: m.cfg.DownloadDir,
		Categorize:  m.cfg.CategorizeOn(),
	}
}

// wholeNumber parses a setting's value and reports what is wrong with it.
// Silently ignoring a value out of range looked identical to accepting one:
// `config segments 999` printed the settings back with segments still at 8
// and said nothing about why.
func wholeNumber(key, v string, lo, hi int) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s needs a whole number, not %q", key, v)
	}
	if n < lo || n > hi {
		return 0, fmt.Errorf("%s must be between %d and %d, not %d", key, lo, hi, n)
	}
	return n, nil
}

// SetSettings applies key/value updates live, persists them, and broadcasts.
func (m *Manager) SetSettings(kv map[string]string) error {
	m.mu.Lock()
	for k, v := range kv {
		switch k {
		case "intercept":
			m.cfg.SetIntercept(v == "true" || v == "on" || v == "1")
		case "categorize":
			m.cfg.SetCategorize(v == "true" || v == "on" || v == "1")
		case "segments":
			n, err := wholeNumber(k, v, 1, 32)
			if err != nil {
				m.mu.Unlock()
				return err
			}
			m.cfg.Segments = n
			m.eng.Segments = n
		case "max-active", "maxActive":
			n, err := wholeNumber(k, v, 1, 10)
			if err != nil {
				m.mu.Unlock()
				return err
			}
			m.cfg.MaxActive = n
		case "min-size", "minSizeMB":
			n, err := wholeNumber(k, v, 0, 1<<20)
			if err != nil {
				m.mu.Unlock()
				return err
			}
			m.cfg.MinSizeMB = n
		case "speed-limit", "speedLimitKB":
			// 0 lifts the cap; anything above 1 GB/s is not a limit worth
			// applying and is almost certainly a typo.
			n, err := wholeNumber(k, v, 0, 1<<20)
			if err != nil {
				m.mu.Unlock()
				return err
			}
			m.cfg.SpeedLimitKB = n
			m.eng.Limiter.SetRate(int64(n) << 10)
		case "dir", "downloadDir":
			if v != "" {
				m.cfg.DownloadDir = v
			}
		default:
			m.mu.Unlock()
			return fmt.Errorf("unknown setting: %s (intercept|categorize|segments|max-active|min-size|speed-limit|dir)", k)
		}
	}
	err := m.cfg.Save()
	m.mu.Unlock()
	m.schedule() // a raised max-active may free slots
	m.broadcast()
	return err
}

// Subscribe registers a watch channel; the returned func unsubscribes.
func (m *Manager) Subscribe() (<-chan *ipc.Snapshot, func()) {
	ch := make(chan *ipc.Snapshot, 4)
	m.mu.Lock()
	m.subs[ch] = struct{}{}
	m.mu.Unlock()
	// Prime with the current state so subscribers render immediately.
	select {
	case ch <- m.Snapshot():
	default:
	}
	return ch, func() {
		m.mu.Lock()
		delete(m.subs, ch)
		m.mu.Unlock()
	}
}

func (m *Manager) broadcast() {
	snap := m.Snapshot()
	m.mu.Lock()
	for ch := range m.subs {
		select {
		case ch <- snap:
		default: // slow subscriber keeps its backlog; next snapshot supersedes
		}
	}
	m.mu.Unlock()
}
