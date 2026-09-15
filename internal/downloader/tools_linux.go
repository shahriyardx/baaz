package downloader

import "context"

// Linux distributions package yt-dlp and ffmpeg properly, and fetching
// binaries behind the package manager's back would be the wrong thing to do
// there. The install hint names the right command instead.
// EnsureMediaTools is a no-op here: Linux gets its tools from the package
// manager, which the installer and the error messages point at.
func (e *Engine) EnsureMediaTools(ctx context.Context, note func(string)) error {
	return nil
}

func (e *Engine) ensureMediaTools(ctx context.Context, note func(string)) error {
	return nil
}
