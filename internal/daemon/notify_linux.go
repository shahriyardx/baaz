package daemon

import "os/exec"

// notify sends a desktop notification; failures are irrelevant.
func notify(title, body string) {
	go exec.Command("notify-send", "-a", "baaz", "-i", "folder-download", title, body).Run()
}
