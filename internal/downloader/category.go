package downloader

import (
	"path/filepath"
	"strings"
)

// IDM-style category folders, keyed by file extension.
var categories = map[string]string{
	"mp4": "Videos", "mkv": "Videos", "webm": "Videos", "avi": "Videos",
	"mov": "Videos", "flv": "Videos", "wmv": "Videos", "m4v": "Videos", "ts": "Videos",

	"mp3": "Music", "m4a": "Music", "flac": "Music", "ogg": "Music",
	"wav": "Music", "opus": "Music", "aac": "Music",

	"pdf": "Documents", "doc": "Documents", "docx": "Documents", "xls": "Documents",
	"xlsx": "Documents", "ppt": "Documents", "pptx": "Documents", "txt": "Documents",
	"epub": "Documents", "odt": "Documents", "csv": "Documents", "md": "Documents",

	"exe": "Programs", "msi": "Programs", "deb": "Programs", "rpm": "Programs",
	"appimage": "Programs", "apk": "Programs", "jar": "Programs", "run": "Programs",

	"zip": "Compressed", "rar": "Compressed", "7z": "Compressed", "tar": "Compressed",
	"gz": "Compressed", "bz2": "Compressed", "xz": "Compressed", "zst": "Compressed",
	"tgz": "Compressed", "iso": "Compressed",

	"jpg": "Images", "jpeg": "Images", "png": "Images", "gif": "Images",
	"webp": "Images", "svg": "Images", "bmp": "Images", "avif": "Images",
}

func CategoryFor(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	if c, ok := categories[ext]; ok {
		return c
	}
	return "Other"
}

// CategorizedDir places a file under <base>/baaz/<Category>.
func CategorizedDir(base, name string) string {
	return filepath.Join(base, "baaz", CategoryFor(name))
}
