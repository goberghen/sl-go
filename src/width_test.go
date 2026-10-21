package main

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func newBufWriter(w io.Writer) *bufio.Writer { return bufio.NewWriter(w) }

func TestRuneWidth(t *testing.T) {
	cases := map[string]int{
		"abc":                3,
		"日本語":                6, // wide: two cells each
		"한국어":                6,
		"🎬":                  2, // emoji
		"ё":                  1,
		"e\u0301":            1, // combining accent
		"\u200b":             0, // zero width space
		"\x1b[31mabc\x1b[0m": 3, // escape sequences are free
		"ﬁ":                  1,
	}
	for s, want := range cases {
		if got := displayLen(s); got != want {
			t.Errorf("displayLen(%q) = %d, want %d", s, got, want)
		}
	}
}

// TestCJKColumns verifies that a wide name does not push the next column.
func TestCJKColumns(t *testing.T) {
	list := []*Entry{
		{Type: "file", Display: "日本語のファイル.txt", Pre: "  ", Style: ""},
		{Type: "file", Display: "short.txt", Pre: "  ", Style: ""},
		{Type: "file", Display: "中文.txt", Pre: "  ", Style: ""},
		{Type: "file", Display: "b.txt", Pre: "  ", Style: ""},
	}
	opts := defaultOptions()
	opts.Color = 0
	var out strings.Builder
	c := &Ctx{opts: opts, w: newBufWriter(&out), errw: &strings.Builder{}, width: 46}
	c.render(list, 0)
	c.w.Flush()

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 rows, got %d: %q", len(lines), out.String())
	}
	// second column must start at the same terminal column in both rows
	start := func(line string) int {
		// the first cell ends at the first run of >=2 spaces after the name
		i := strings.Index(line, "  ")
		for i >= 0 && i+2 < len(line) {
			if strings.HasPrefix(line[i:], "  ") {
				// trim to the start of the next non-space run
				j := i
				for j < len(line) && line[j] == ' ' {
					j++
				}
				return displayLen(line[:j])
			}
			i = strings.Index(line[i+1:], "  ")
			if i >= 0 {
				i++
			}
		}
		return -1
	}
	if a, b := start(lines[0]), start(lines[1]); a != b || a < 0 {
		t.Errorf("second column starts at %d and %d, want the same:\n%s", a, b, out.String())
	}
}
