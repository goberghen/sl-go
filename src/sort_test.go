package main

// Tests for --sort. The rules under test are the ones the documentation
// promises: largest first for size, newest first for time (mtime) and atime,
// directories left in name order under --sort size, --prefix turned off for any
// order other than name, and --sort size implying --relsize 100.

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// sortRender lists dir with the given options and returns the names from the
// given set in the order they were printed. Group titles and marks are ignored,
// a directory printed as "src/3" is recognised as "src".
func sortRender(t *testing.T, dir string, known []string, tweak func(*Options)) []string {
	t.Helper()
	var buf bytes.Buffer
	opts := defaultOptions()
	opts.Color = 0
	opts.One = true
	opts.Summary = 0
	opts.Expando = 0
	if tweak != nil {
		tweak(opts)
	}
	c := &Ctx{opts: opts, w: bufio.NewWriter(&buf), errw: &buf, width: 100, now: time.Now()}
	c.list(dir, nil, 0)
	c.w.Flush()
	return pickNames(buf.String(), known)
}

// pickNames pulls the given names out of a one-per-line listing, in order.
// Longer names are tried first so that "aaa.txt" wins over "aaa".
func pickNames(out string, known []string) []string {
	byLen := append([]string(nil), known...)
	for i := 0; i < len(byLen); i++ {
		for j := i + 1; j < len(byLen); j++ {
			if len(byLen[j]) > len(byLen[i]) {
				byLen[i], byLen[j] = byLen[j], byLen[i]
			}
		}
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		var hit string
		for _, k := range byLen {
			if strings.Contains(line, k) {
				hit = k
				break
			}
		}
		if hit == "" {
			continue
		}
		if n := len(names); n > 0 && names[n-1] == hit {
			continue
		}
		names = append(names, hit)
	}
	return names
}

func namesIn(names []string, want ...string) bool {
	if len(names) != len(want) {
		return false
	}
	for i := range want {
		if names[i] != want[i] {
			return false
		}
	}
	return true
}

func TestSortSizeIgnoresDirectories(t *testing.T) {
	dir := t.TempDir()
	// files: 3000 > 100 > 50 bytes
	for name, size := range map[string]int{"aaa.txt": 50, "mmm.txt": 3000, "zzz.txt": 100} {
		if err := os.WriteFile(filepath.Join(dir, name), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// directories: "aaa" has one entry, "zzz" has five, so sorting them by
	// their Size (a count, not bytes) would put zzz first
	if err := os.MkdirAll(filepath.Join(dir, "aaa"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "aaa", "one"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "zzz"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"1", "2", "3", "4", "5"} {
		if err := os.WriteFile(filepath.Join(dir, "zzz", n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	known := []string{"aaa", "zzz", "aaa.txt", "mmm.txt", "zzz.txt"}
	got := sortRender(t, dir, known, func(o *Options) { o.SortMode = "size" })
	// dir group first (no title, name order), then doc in size order
	if !namesIn(got, "aaa", "zzz", "mmm.txt", "zzz.txt", "aaa.txt") {
		t.Errorf("--sort size: directories must stay in name order and files go largest first, got %v", got)
	}

	byName := sortRender(t, dir, known, nil)
	if !namesIn(byName, "aaa", "zzz", "aaa.txt", "mmm.txt", "zzz.txt") {
		t.Errorf("default order changed: %v", byName)
	}
}

func TestSortTimeAndAtime(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"f1.txt", "f2.txt", "f3.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// mtime order: f2 > f3 > f1 ; atime order: f1 > f3 > f2 (on purpose different)
	mt := map[string]string{"f1.txt": "2024-01-01", "f2.txt": "2024-03-01", "f3.txt": "2024-02-01"}
	at := map[string]string{"f1.txt": "2024-06-01", "f2.txt": "2024-04-01", "f3.txt": "2024-05-01"}
	for n := range mt {
		m, err := time.Parse("2006-01-02", mt[n])
		if err != nil {
			t.Fatal(err)
		}
		a, err := time.Parse("2006-01-02", at[n])
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(filepath.Join(dir, n), a, m); err != nil {
			t.Fatal(err)
		}
	}

	known := []string{"f1.txt", "f2.txt", "f3.txt"}
	if got := sortRender(t, dir, known, func(o *Options) { o.SortMode = "time" }); !namesIn(got, "f2.txt", "f3.txt", "f1.txt") {
		t.Errorf("--sort time must be newest mtime first, got %v", got)
	}
	if got := sortRender(t, dir, known, func(o *Options) { o.SortMode = "atime" }); !namesIn(got, "f1.txt", "f3.txt", "f2.txt") {
		t.Errorf("--sort atime must be newest atime first, got %v", got)
	}
}

func TestSortImpliesRelsize(t *testing.T) {
	cases := []struct {
		argv []string
		want int
	}{
		{[]string{"--sort", "size"}, 100},
		{[]string{"--sort=size"}, 100},
		{[]string{"--sort", "size", "--relsize", "20"}, 20},
		{[]string{"--relsize", "20", "--sort", "size"}, 20},
		{[]string{"--sort", "size", "-o"}, 0},
		{[]string{"--sort", "time"}, 5},
		{nil, 5},
	}
	for _, c := range cases {
		o, _, err := parseArgs(c.argv)
		if err != nil {
			t.Fatalf("%v: %v", c.argv, err)
		}
		applyImplications(o)
		if o.RelSize != c.want {
			t.Errorf("%v: RelSize = %d, want %d", c.argv, o.RelSize, c.want)
		}
	}

	if _, _, err := parseArgs([]string{"--sort", "bogus"}); err == nil {
		t.Error("--sort bogus must be rejected")
	}
}

func TestSortTurnsPrefixElisionOff(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"myproject-notes.txt", "myproject-plan.txt", "myproject-todo.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("xxxxxxxxxx"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw := func(tweak func(*Options)) string {
		var buf bytes.Buffer
		opts := defaultOptions()
		opts.Color, opts.One, opts.Summary, opts.Expando = 0, true, 0, 0
		opts.Prefix = 4
		tweak(opts)
		c := &Ctx{opts: opts, w: bufio.NewWriter(&buf), errw: &buf, width: 100, now: time.Now()}
		c.list(dir, nil, 0)
		c.w.Flush()
		return buf.String()
	}

	sorted := raw(func(o *Options) { o.SortMode = "size" })
	for _, n := range []string{"myproject-notes.txt", "myproject-plan.txt", "myproject-todo.txt"} {
		if !strings.Contains(sorted, n) {
			t.Errorf("with --sort size the prefix must not be elided, %q missing:\n%s", n, sorted)
		}
	}

	// sanity: with name order the same listing does elide it, but only for the
	// second and later entries of the group -- the first one has to show the
	// prefix the others are being cut against
	byName := raw(func(o *Options) {})
	if !strings.Contains(byName, "-todo.txt") || strings.Contains(byName, "myproject-todo.txt") {
		t.Errorf("--prefix 4 should have elided the shared prefix in name order:\n%s", byName)
	}
}

func TestSortFlatNogroup(t *testing.T) {
	dir := t.TempDir()
	for name, size := range map[string]int{"small.txt": 10, "big.txt": 4000, "mid.txt": 500} {
		if err := os.WriteFile(filepath.Join(dir, name), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	known := []string{"sub", "big.txt", "mid.txt", "small.txt"}
	got := sortRender(t, dir, known, func(o *Options) { o.SortMode = "size"; o.Group, o.GroupTitle = false, false })
	if !namesIn(got, "sub", "big.txt", "mid.txt", "small.txt") {
		t.Errorf("--nogroup --sort size: directories first in name order, then by size, got %v", got)
	}
}

var sizeMark = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?[BKMG]$`)

// TestRelsize100ShowsEverySize pins what --relsize 100 means, because
// --sort size depends on it: the order is only readable if the numbers it
// orders by are all on screen. Below 100 the cutoff is clamped to the median so
// that "everything created a minute apart" does not light up, but at 100 the
// user asked for every value.
func TestRelsize100ShowsEverySize(t *testing.T) {
	dir := t.TempDir()
	sizes := map[string]int{
		"big.txt": 600000, "mid.txt": 300000, "one.txt": 200000,
		"two.txt": 90000, "three.txt": 30000, "four.txt": 5000, "five.txt": 1,
	}
	for n, size := range sizes {
		if err := os.WriteFile(filepath.Join(dir, n), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	count := func(pct int) int {
		var buf bytes.Buffer
		opts := defaultOptions()
		opts.Color, opts.One, opts.Summary, opts.Expando = 0, true, 0, 0
		opts.RelSize = pct
		c := &Ctx{opts: opts, w: bufio.NewWriter(&buf), errw: &buf, width: 100, now: time.Now()}
		c.list(dir, nil, 0)
		c.w.Flush()
		n := 0
		for _, line := range strings.Split(buf.String(), "\n") {
			// a size mark is a field such as 586K, 4.9K or 1B; other marks
			// (0, <NOW, -2y, ->) never look like that
			for _, f := range strings.Fields(line) {
				if sizeMark.MatchString(f) {
					n++
					break
				}
			}
		}
		return n
	}

	if got := count(100); got != len(sizes) {
		t.Errorf("--relsize 100 marked %d of %d sizes, want all", got, len(sizes))
	}
	if got := count(20); got >= len(sizes) {
		t.Errorf("--relsize 20 marked %d sizes, that is everything and not 20%%", got)
	}
}
