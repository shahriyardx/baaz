package daemon

import (
	"os/exec"
	"strings"
)

// notify sends a Notification Center banner; failures are irrelevant.
//
// osascript is the only notification path that needs no extra install. The
// first banner asks the user to allow notifications for the script runner —
// a one-time macOS prompt, not an error.
func notify(title, body string) {
	script := "display notification " + asQuote(body) +
		" with title " + asQuote(title)
	go exec.Command("osascript", "-e", script).Run()
}

// asQuote renders a Go string as an AppleScript string literal. Filenames
// reach here unfiltered, so quotes and backslashes must not break out of the
// literal and turn a download name into script.
func asQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	// AppleScript literals cannot span lines; collapse any newlines.
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return `"` + r.Replace(s) + `"`
}
