package daemon

import "testing"

// Download names come from the server and can hold anything. A name that
// closes the AppleScript literal would turn a notification into script
// execution, so quoting is the boundary that has to hold.
func TestAsQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{`plain.zip`, `"plain.zip"`},
		{`say "hi".mp4`, `"say \"hi\".mp4"`},
		{`back\slash.bin`, `"back\\slash.bin"`},
		{"line\nbreak.txt", `"line break.txt"`},
		{"carriage\rreturn.txt", `"carriage return.txt"`},
		{`" & (do shell script "id") & "`, `"\" & (do shell script \"id\") & \""`},
		{``, `""`},
	}
	for _, c := range cases {
		if got := asQuote(c.in); got != c.want {
			t.Errorf("asQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
