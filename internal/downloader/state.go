package downloader

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Save persists the job atomically (tmp + rename). State files hold cookies,
// so they are user-readable only.
func Save(dir string, j *Job) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	j.mu.Lock()
	data, err := json.MarshalIndent(j, "", "  ")
	j.mu.Unlock()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, j.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Remove(dir string, j *Job) error {
	return os.Remove(filepath.Join(dir, j.ID+".json"))
}

// LoadAll reads every job state file; unreadable files are skipped.
func LoadAll(dir string) []*Job {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var jobs []*Job
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var j Job
		if err := json.Unmarshal(data, &j); err != nil {
			continue
		}
		jobs = append(jobs, &j)
	}
	return jobs
}
