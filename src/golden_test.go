package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildFixture plants a small tree with fixed sizes and mtimes so that the
// listing is reproducible.
func buildFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := time.Now().Add(-90 * 24 * time.Hour)
	files := map[string]int{
		"README":         10,
		"Makefile":       20,
		"main.c":         100,
		"notes.md":       30,
		"empty.txt":      0,
		"cover.png":      4000,
		"song.mp3":       200000,
		"archive.tar.gz": 5000,
		".hidden":        5,
		"junk.bak":       7,
		"episode 01.mkv": 1000,
		"episode 02.mkv": 1000,
		"episode 03.mkv": 1000,
		"episode 04.mkv": 1000,
		"episode 05.mkv": 1000,
		"episode 06.mkv": 1000,
		"a long file name that will have to be shortened somewhere.txt": 12,
	}
	for name, size := range files {
		if err := os.WriteFile(filepath.Join(dir, name), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"one.c", "two.c"} {
		if err := os.WriteFile(filepath.Join(dir, "src", n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// make everything look 90 days old, so no "recent" markers appear
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		os.Chtimes(filepath.Join(dir, e.Name()), old, old)
	}
	os.Chtimes(filepath.Join(dir, "src", "one.c"), old, old)
	os.Chtimes(filepath.Join(dir, "src", "two.c"), old, old)
	return dir
}

func TestGoldenListing(t *testing.T) {
	dir := buildFixture(t)

	var buf bytes.Buffer
	opts := defaultOptions()
	opts.Color = 0
	c := &Ctx{opts: opts, w: bufio.NewWriter(&buf), errw: &buf, width: 80, now: time.Now()}
	c.list(dir, nil, 0)
	c.w.Flush()

	want := strings.TrimSpace(`
  build
  Makefile <90d
  README
  dev
  main.c
  src/2
  doc
  empty.txt0
  a long file name that will have to be shortened somewhere.txt
  image
  cover.png
  audio
   song.mp3 195K
  video
  episode 01.mkv...6
  other
  notes.md
  archive
  archive.tar.gz
  15 files (210K) + 1 .dotfile + 1 ignored, 1 directory (2 files)
`)
	got := strings.TrimSpace(buf.String())
	if got != want {
		t.Errorf("listing mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	t.Logf("\n%s", buf.String())
}

// TestA2Group guards the Apple II group: it must survive any refactoring.
// The original script has a whole group for disk images (2012), including the
// ".nib of exactly 232960 bytes" special case that tells a 5.25" nibble image
// from a NeXT nib bundle. The Go port additionally knows the formats that
// appeared later (.woz, .hdv, .d13, ...) and the 13-sector .nib size.
func TestA2Group(t *testing.T) {
	dir := t.TempDir()
	mk := func(name string, size int) {
		if err := os.WriteFile(filepath.Join(dir, name), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// the original's own list
	mk("Karateka.dsk", 143360)
	mk("Oregon Trail.do", 143360)
	mk("ProDOS.po", 143360)
	mk("Ultima.2mg", 200000)
	mk("Backup.shk", 60000)
	mk("Zork.nib", 232960) // 35 * 6656: nibble image of a 5.25" disk
	// not a disk image
	mk("NotADisk.nib", 4096) // a NeXT/IB nib, stays in dev
	// formats added by the port
	mk("Wizardry.woz", 200000) // 2018 flux image
	mk("Volume.hdv", 819200)   // ProDOS hard disk
	mk("DOS32.d13", 116480)    // 13-sector DOS 3.2
	mk("DOS32b.nib", 116480)   // ... and its nibble form
	mk("Applesauce.edsk", 200000)

	groupOf := func(name string) string {
		opts := defaultOptions()
		c := &Ctx{opts: opts, w: bufio.NewWriter(&bytes.Buffer{}), errw: &bytes.Buffer{}, width: 80, now: time.Now()}
		e := c.buildEntry(dir, name, filepath.Join(dir, name), map[string]bool{name: true},
			&counters{}, new([]string))
		if e == nil {
			t.Fatalf("%s: entry was filtered out", name)
		}
		return e.Group
	}

	for _, n := range []string{"Karateka.dsk", "Oregon Trail.do", "ProDOS.po", "Ultima.2mg", "Backup.shk",
		"Zork.nib", "Wizardry.woz", "Volume.hdv", "DOS32.d13", "DOS32b.nib", "Applesauce.edsk"} {
		if g := groupOf(n); g != "A2" {
			t.Errorf("%s landed in group %q, want A2", n, g)
		}
	}
	if g := groupOf("NotADisk.nib"); g != "dev" {
		t.Errorf("a 4096-byte .nib must stay in dev (NeXT bundle), got %q", g)
	}
}

// TestInlineSingleton covers --inline-singleton, which finishes the branch the
// original had disabled ("elseif {0 && $size==1}").
func TestInlineSingleton(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "proj", "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proj", "src", "main.c"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	render := func(inline bool) string {
		var buf bytes.Buffer
		opts := defaultOptions()
		opts.Color = 0
		opts.InlineOne = inline
		c := &Ctx{opts: opts, w: bufio.NewWriter(&buf), errw: &buf, width: 80, now: time.Now()}
		c.list(dir, nil, 0)
		c.w.Flush()
		return buf.String()
	}

	if got := render(false); !strings.Contains(got, "proj/") {
		t.Errorf("without the flag the directory keeps its mnemonic count:\n%s", got)
	}
	got := render(true)
	if !strings.Contains(got, "proj/src/main.c") {
		t.Errorf("with --inline-singleton the lone child should be shown:\n%s", got)
	}
}

// TestSeriesWithMarks is the end-to-end counterpart of TestSeriesCollapses:
// the unit test uses entries without marks, so it could not catch the rule that
// actually decides whether a run is collapsed in a real listing. A marked entry
// breaks the run (this is the Tcl original's rule, see --series), so six fresh
// episodes stay six lines -- until the clock moves on.
func TestSeriesWithMarks(t *testing.T) {
	dir := t.TempDir()
	for i := 1; i <= 6; i++ {
		name := filepath.Join(dir, "episode 0"+string(rune('0'+i))+".mkv")
		if err := os.WriteFile(name, bytes.Repeat([]byte("x"), 1000), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	list := func() string {
		var buf bytes.Buffer
		opts := defaultOptions()
		opts.Color = 0
		opts.Expando = 0
		c := &Ctx{opts: opts, w: bufio.NewWriter(&buf), errw: &buf, width: 100, now: time.Now()}
		c.list(dir, nil, 0)
		c.w.Flush()
		return buf.String()
	}

	fresh := list()
	if strings.Contains(fresh, "...") {
		t.Errorf("fresh files are marked, so the run must not collapse:\n%s", fresh)
	}
	for i := 1; i <= 6; i++ {
		name := fmt.Sprintf("episode 0%d.mkv", i)
		if !strings.Contains(fresh, name) {
			t.Errorf("%s missing from the listing:\n%s", name, fresh)
		}
	}

	// same files, old timestamps: no relative mark any more, so they collapse
	old := time.Now().Add(-400 * 24 * time.Hour)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if err := os.Chtimes(filepath.Join(dir, e.Name()), old, old); err != nil {
			t.Fatal(err)
		}
	}
	aged := list()
	if !strings.Contains(aged, "episode 01.mkv") || !strings.Contains(aged, "...6") {
		t.Errorf("old files should collapse into \"episode 01.mkv...6\":\n%s", aged)
	}
	if strings.Contains(aged, "episode 02.mkv") {
		t.Errorf("collapsed run should not list the inner names:\n%s", aged)
	}
}

// TestNoTrailingSpaces guards the column filler. bump() inserts nil
// placeholders to push a group title out of the bottom cell, and a placeholder
// that lands last in a row used to leave spaces at the end of the line (visible
// with "cat -A", annoying in pipes). The name list is the one that reproduced
// it: nine entries in five groups at 80 columns.
func TestNoTrailingSpaces(t *testing.T) {
	dir := t.TempDir()
	spec := map[string]int{
		"a-7.log": 0, "bb-1.sh": 0, "dddd-8.mkv": 0, "ffffff-2.log": 0,
		"ccc-0.zip": 0, "bb-6.sh": 10, "google-5.md": 10,
		"a-4.md": 900000, "data-3.sh": 10,
	}
	for n, size := range spec {
		if err := os.WriteFile(filepath.Join(dir, n), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	for _, only := range []bool{false, true} {
		for width := 40; width <= 120; width += 2 {
			var buf bytes.Buffer
			opts := defaultOptions()
			opts.Color = 0
			opts.Only = only
			opts.Summary = 0
			c := &Ctx{opts: opts, w: bufio.NewWriter(&buf), errw: &buf, width: width, now: time.Now()}
			c.list(dir, nil, 0)
			c.w.Flush()
			for _, line := range strings.Split(buf.String(), "\n") {
				if strings.HasSuffix(line, " ") {
					t.Errorf("width %d, only=%v: trailing space in %q", width, only, line)
				}
			}
		}
	}
}
