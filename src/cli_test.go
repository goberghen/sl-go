package main

import (
	"strings"
	"testing"
)

// TestOptionStyle: one dash means a short option, two dashes mean a long one.
// The Tcl version's "-width 100" was removed deliberately (it turned a typo
// into a silent misparse); this test keeps it removed and keeps the error
// message useful.
func TestOptionStyle(t *testing.T) {
	ok := [][]string{
		{"--only"}, {"--width=100"}, {"--width", "100"},
		{"-w", "100"}, {"-w100"}, {"-a"}, {"-1"}, {"-o"},
		{"-c", "--color=never"}, {"--rule", `\.tmp$ -> ignore`},
	}
	for _, argv := range ok {
		o, _, err := parseArgs(argv)
		if err != nil {
			t.Errorf("%v: unexpected error: %v", argv, err)
			continue
		}
		if o == nil {
			t.Errorf("%v: no options", argv)
		}
	}

	bad := map[string]string{
		"-only":      "did you mean --only?",
		"-width":     "did you mean --width?",
		"-width=100": "did you mean --width?",
		"-series=3":  "did you mean --series?",
		"-x":         "unknown option: -x",
	}
	for arg, want := range bad {
		_, _, err := parseArgs([]string{arg})
		if err == nil {
			t.Errorf("%s: accepted, but one dash may only spell a short option", arg)
			continue
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %q, want it to contain %q", arg, err, want)
		}
	}

	// the removed spelling must not be silently reinterpreted as short flags
	_, _, err := parseArgs([]string{"-only"})
	if err != nil && strings.Contains(err.Error(), "--only") {
		return
	}
	t.Errorf("the message for -only should point at --only, got %v", err)
}
