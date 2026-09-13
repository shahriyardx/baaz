package downloader

// ytdlpBinDirs are the directories package managers install yt-dlp (and
// ffmpeg, which yt-dlp needs to merge video with audio) into on Linux.
var ytdlpBinDirs = []string{
	"/usr/bin",
	"/usr/local/bin",
	"/var/lib/flatpak/exports/bin",
}

const ytdlpInstallHint = "pacman -S yt-dlp / apt install yt-dlp / dnf install yt-dlp"
