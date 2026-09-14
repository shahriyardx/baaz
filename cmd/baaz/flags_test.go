package main

import (
	"flag"
	"reflect"
	"testing"
)

func addFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.String("out", "", "")
	fs.String("format", "", "")
	fs.Bool("quiet", false, "")
	return fs
}

// The usage line documents `baaz add URL --out NAME`, which Go's flag package
// refuses because it stops at the first positional word.
func TestFlagsAfterTheURLAreAccepted(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"flag after positional", []string{"http://x", "--out", "n"}, []string{"--out", "n", "http://x"}},
		{"flag before positional", []string{"--out", "n", "http://x"}, []string{"--out", "n", "http://x"}},
		{"equals form", []string{"http://x", "--out=n"}, []string{"--out=n", "http://x"}},
		{"two flags after", []string{"http://x", "--out", "n", "--format", "480"},
			[]string{"--out", "n", "--format", "480", "http://x"}},
		{"bool flag keeps the next word positional", []string{"--quiet", "http://x"}, []string{"--quiet", "http://x"}},
		{"bool flag after positional", []string{"http://x", "--quiet"}, []string{"--quiet", "http://x"}},
		{"bare double dash", []string{"--out", "n", "--", "-weird-name"}, []string{"--out", "n", "-weird-name"}},
		{"no flags", []string{"http://x"}, []string{"http://x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reorderFlags(addFlagSet(), c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("reorderFlags(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// What actually matters: after reordering, the flag set parses one positional.
func TestReorderedArgsParseToOneURL(t *testing.T) {
	for _, in := range [][]string{
		{"http://x", "--out", "n", "--format", "480"},
		{"--out", "n", "http://x", "--format", "480"},
		{"--out", "n", "--format", "480", "http://x"},
	} {
		fs := addFlagSet()
		if err := fs.Parse(reorderFlags(fs, in)); err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if fs.NArg() != 1 || fs.Arg(0) != "http://x" {
			t.Errorf("%q: got %d positionals %q, want just the URL", in, fs.NArg(), fs.Args())
		}
		if fs.Lookup("out").Value.String() != "n" || fs.Lookup("format").Value.String() != "480" {
			t.Errorf("%q: flags lost: out=%q format=%q", in,
				fs.Lookup("out").Value, fs.Lookup("format").Value)
		}
	}
}
