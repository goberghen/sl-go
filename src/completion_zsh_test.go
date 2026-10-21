package main

// Zsh completion, tested the way a user uses it: a real interactive zsh in a
// pty, a real TAB press, and the bytes that end up on the terminal.
//
// This is not paranoia. A completion file can be perfect text and still do
// nothing: it has to sit in a directory that is in $fpath. Debian keeps that
// directory out of the default $fpath for third-party packages (the packaged
// one is /usr/share/zsh/vendor-completions, not /usr/share/zsh/site-functions),
// and a file in the wrong place fails silently -- `sl -<TAB>` completes
// filenames as if no completion existed. Static checks cannot see that; typing
// TAB can. TestDebZshCompletionPath pins the Debian path, this test pins the
// behaviour.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// zshComplete types line, presses TAB and returns what the terminal received.
func zshComplete(t *testing.T, line string) string {
	t.Helper()

	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	if strings.ContainsAny(line, `'"`) {
		t.Fatalf("zshComplete: %q contains a quote, the harness cannot pass it", line)
	}

	// The committed completion file, in a directory that plays the role of
	// $fpath. The test deliberately uses the shipped file, not a fresh one.
	data, err := os.ReadFile(filepath.Join("..", "completions", "_sl"))
	if err != nil {
		t.Fatalf("completions/_sl: %v (run `make completions`)", err)
	}
	fpath := t.TempDir()
	if err := os.WriteFile(filepath.Join(fpath, "_sl"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	// zsh/zpty is zsh's own pty module: it starts a real interactive shell,
	// so compinit, ZLE and the completion widget all behave normally.
	// LINES/COLUMNS are set explicitly: with the default 24 lines zsh stops at
	// "do you wish to see all N possibilities?" instead of printing the list,
	// and the descriptions -- which is what this test is about -- never arrive.
	script := fmt.Sprintf(`zmodload zsh/zpty || exit 3
zpty z %q
zpty -w z "PS1='@> '; LINES=60; COLUMNS=200; export LINES COLUMNS"
zpty -w z 'fpath=(%s $fpath)'
zpty -w z 'autoload -Uz compinit; compinit -u'
zpty -w z "zstyle ':completion:*' list-prompt ''"
zpty -w z 'unsetopt zle_bracketed_paste 2>/dev/null'
sleep 1
zpty -w -n z '%s'$'\t'
sleep 2
buf=
while zpty -r -t z chunk 2>/dev/null; do buf+=$chunk; done
print -r -- "$buf"
`, zsh, fpath, line)

	harness := filepath.Join(t.TempDir(), "harness.zsh")
	if err := os.WriteFile(harness, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	// The harness is timing based: a loaded machine can make zsh print the list
	// after the last read. Run it up to three times and keep the longest output
	// rather than reporting a failure that says nothing about the product.
	best := ""
	for attempt := 0; attempt < 3; attempt++ {
		out, err := exec.Command(zsh, "-f", harness).CombinedOutput()
		if err != nil {
			t.Fatalf("zsh harness: %v\n%s", err, out)
		}
		if len(out) > len(best) {
			best = string(out)
		}
		if strings.Contains(best, "--") {
			break
		}
	}
	return best
}

// TestZshCompletionListsOptions: "sl -" + TAB must offer the options with their
// descriptions (the way dpkg's completion does), not filenames.
func TestZshCompletionListsOptions(t *testing.T) {
	t.Run("sl -", func(t *testing.T) {
		out := zshComplete(t, "sl -")
		for _, want := range []string{
			"--color", "--width", "--rule", "-w", "-1",
			// the description half of the listing
			"show only entries that carry some signal",
			"one flat sorted list, no group titles",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("sl -<TAB> did not offer %q\n%s", want, out)
			}
		}
	})

	t.Run("sl --wid", func(t *testing.T) {
		// One match: zsh inserts it and, because --width takes a value in the
		// next word, leaves the cursor after a space.
		out := zshComplete(t, "sl --wid")
		if !strings.Contains(out, "--width ") {
			t.Errorf("sl --wid<TAB> did not complete to --width\n%s", out)
		}
	})

	t.Run("sl --color=", func(t *testing.T) {
		// An attached value completes to the three choices.
		out := zshComplete(t, "sl --color=")
		for _, want := range []string{"auto", "always", "never"} {
			if !strings.Contains(out, want) {
				t.Errorf("sl --color=<TAB> did not offer %q\n%s", want, out)
			}
		}
	})

	t.Run("sl --no", func(t *testing.T) {
		out := zshComplete(t, "sl --no")
		for _, want := range []string{
			"--nogroup", "one flat sorted list, no group titles",
			"--no-notable", "do not highlight README/TODO/Makefile",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("sl --no<TAB> did not offer %q\n%s", want, out)
			}
		}
	})
}

// TestDebZshCompletionPath pins the one path that is not the obvious one.
func TestDebZshCompletionPath(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "packaging", "build-deb.sh"))
	if err != nil {
		t.Fatal(err)
	}
	deb := string(data)
	if !strings.Contains(deb, "/usr/share/zsh/vendor-completions/_sl") {
		t.Error("build-deb.sh does not install _sl into /usr/share/zsh/vendor-completions, " +
			"which is the only zsh completion directory Debian reads")
	}
	if strings.Contains(deb, "/usr/share/zsh/site-functions/_sl") {
		t.Error("build-deb.sh still installs _sl into /usr/share/zsh/site-functions, " +
			"which is not in Debian's $fpath -- zsh ignores it silently")
	}
}
