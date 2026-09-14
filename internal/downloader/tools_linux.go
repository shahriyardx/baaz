package downloader

import "context"

// Linux distributions package yt-dlp and ffmpeg properly, and fetching
// binaries behind the package manager's back would be the wrong thing to do
// there. The install hint names the right command instead.
func (e *Engine) ensureMediaTools(ctx context.Context, note func(string)) error {
	return nil
}
