package downloader

// ytdlpBinDirs are the directories package managers install yt-dlp (and
// ffmpeg, which yt-dlp needs to merge video with audio) into on macOS.
var ytdlpBinDirs = []string{
	"/opt/homebrew/bin", // Homebrew, Apple Silicon
	"/usr/local/bin",    // Homebrew on Intel, and manual installs
	"/opt/local/bin",    // MacPorts
}

const ytdlpInstallHint = "brew install yt-dlp ffmpeg"
