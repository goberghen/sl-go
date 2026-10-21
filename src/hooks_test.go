package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAnnotateBatch covers --annotate-batch: one command per directory, the
// paths on stdin, one answer line per path. The interesting parts are that the
// answers land on the right entries (a line protocol, not a per-entry call) and
// that the command really runs once, not once per name.
func TestAnnotateBatch(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runs := filepath.Join(t.TempDir(), "runs")
	t.Setenv("SL_TEST_RUNS", runs)

	// one line per path, derived from the path itself, and a record of each run
	cmd := `echo run >> "$SL_TEST_RUNS"; sed 's|^.*/||'`

	var out, errw bytes.Buffer
	o := defaultOptions()
	o.Color = 0
	o.AnnotateBatch = cmd
	c := &Ctx{opts: o, w: bufio.NewWriter(&out), errw: &errw, width: 80, now: time.Now()}
	if err := c.prepareHooks(); err != nil {
		t.Fatalf("prepareHooks: %v", err)
	}
	c.list(dir, nil, 0)
	c.w.Flush()

	got := out.String()
	for _, want := range []string{"a.go a.go", "b.go b.go", "c.txt c.txt"} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
	if errw.Len() != 0 {
		t.Errorf("unexpected stderr: %s", errw.String())
	}

	rec, err := os.ReadFile(runs)
	if err != nil {
		t.Fatalf("the command never ran: %v", err)
	}
	if n := strings.Count(string(rec), "run"); n != 1 {
		t.Errorf("command ran %d times for one directory, want 1", n)
	}
}

// TestAnnotateBatchMatch: --annotate-match must filter what is sent to the
// command, so that the answer lines stay aligned with the input.
func TestAnnotateBatchMatch(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.txt"} {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644)
	}
	var out, errw bytes.Buffer
	o := defaultOptions()
	o.Color = 0
	o.AnnotateMatch = `\.go$`
	o.AnnotateBatch = `sed 's|^.*/||'`
	c := &Ctx{opts: o, w: bufio.NewWriter(&out), errw: &errw, width: 80, now: time.Now()}
	if err := c.prepareHooks(); err != nil {
		t.Fatalf("prepareHooks: %v", err)
	}
	c.list(dir, nil, 0)
	c.w.Flush()

	if !strings.Contains(out.String(), "a.go a.go") {
		t.Errorf("matched entry not annotated:\n%s", out.String())
	}
	if strings.Contains(out.String(), "b.txt b.txt") {
		t.Errorf("unmatched entry annotated:\n%s", out.String())
	}
}

// TestAnnotateBatchMismatch: a command that answers with the wrong number of
// lines is a bug in the command. sl says so once and annotates what it can; it
// must not attach answers to the wrong names.
func TestAnnotateBatchMismatch(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.go", "b.go"} {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644)
	}
	var out, errw bytes.Buffer
	o := defaultOptions()
	o.Color = 0
	o.AnnotateBatch = `echo single`
	c := &Ctx{opts: o, w: bufio.NewWriter(&out), errw: &errw, width: 80, now: time.Now()}
	if err := c.prepareHooks(); err != nil {
		t.Fatalf("prepareHooks: %v", err)
	}
	c.list(dir, nil, 0)
	c.w.Flush()

	if !strings.Contains(errw.String(), "1 lines for 2 paths") {
		t.Errorf("no warning about the line count, stderr = %q", errw.String())
	}
	if n := strings.Count(out.String(), "single"); n != 0 {
		t.Errorf("annotations kept %d times after a line-count mismatch, want 0 (they would sit on the wrong names)", n)
	}
}

// TestAnnotateOptionsClash: the two hooks cannot be combined, because they would
// fight over the same text.
func TestAnnotateOptionsClash(t *testing.T) {
	o := defaultOptions()
	o.Annotate = "echo a"
	o.AnnotateBatch = "echo b"
	c := &Ctx{opts: o}
	err := c.prepareHooks()
	if err == nil {
		t.Fatal("--annotate with --annotate-batch must be an error")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("unexpected error: %v", err)
	}
}
