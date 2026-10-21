package main

// The man page is the reference the user reads when the binary is already
// installed, so every option the binary accepts has to be in it. This test
// exists because six of them were missing until the string-check change made
// someone look: --autosearch, --notable, --no-notable, --shorten, --k1024 and
// --startup.
//
// The check is textual on purpose: the roff source writes "--flag" as
// "\-\-flag", so searching for the escaped spelling catches a documented
// option and nothing else.

import (
	"os"
	"strings"
	"testing"
)

func TestManPageDocumentsEveryOption(t *testing.T) {
	man, err := os.ReadFile("../docs/sl.1")
	if err != nil {
		t.Fatalf("read man page: %v", err)
	}
	text := string(man)
	for _, s := range optionSpecs {
		needle := `\-\-` + s.long
		if !strings.Contains(text, needle) {
			t.Errorf("docs/sl.1 does not document --%s", s.long)
		}
	}
}

// TestManPageRendersPatterns guards the escape that ate a backslash once:
// "(?i)\\W..." in the source has to render as "(?i)\W...", so the roff file must
// use the \e escape rather than a doubled backslash.
func TestManPageRendersPatterns(t *testing.T) {
	man, err := os.ReadFile("../docs/sl.1")
	if err != nil {
		t.Fatalf("read man page: %v", err)
	}
	if !strings.Contains(string(man), `\eW(urgent|password|pw|live)\eW`) {
		t.Error(`docs/sl.1: the autosearch default must be written with \e escapes`)
	}
}
