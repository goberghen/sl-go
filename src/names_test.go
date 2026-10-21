package main

import (
	"sort"
	"strings"
	"testing"
)

func TestTclExtRoot(t *testing.T) {
	cases := []struct{ path, ext, root string }{
		{"foo.txt", ".txt", "foo"},
		{"dir/foo.tar.gz", ".gz", "dir/foo.tar"},
		{".bashrc", "", ".bashrc"},
		{"noext", "", "noext"},
		{"a.b.c", ".c", "a.b"},
		{"dir.d/file", "", "dir.d/file"},
	}
	for _, c := range cases {
		if got := tclExt(c.path); got != c.ext {
			t.Errorf("tclExt(%q) = %q, want %q", c.path, got, c.ext)
		}
		if got := tclRoot(c.path); got != c.root {
			t.Errorf("tclRoot(%q) = %q, want %q", c.path, got, c.root)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		dir, name, etype string
		group            string
		level            int
		sfx              string
	}{
		{".", "photo.JPG", "file", "image", 4, ".jpg"},
		{".", "notes.txt", "file", "dev", 4, ".txt"}, // "notes" is a dev name
		{".", "main.c", "file", "dev", 4, ".c"},
		{".", "header.h", "file", "dev", 3, ".h"}, // -.h: hidden until -a
		{".", "README", "file", "build", 4, ""},
		{".", "README.md", "file", "other", 4, ".md"}, // exact name only, not a prefix
		{".", "movie.mkv", "file", "video", 4, ".mkv"},
		{".", "movie.mkv.gz", "file", "video", 4, ".mkv"}, // the wrapper is stripped before matching
		{".", ".bashrc", "file", "dotfile", 2, ""},
		{".", "junk.bak", "file", "ignore", 1, ".bak"},
		{".", "foo~", "file", "ignore", 1, ""},
		{".", "subdir", "directory", "other", 4, ""},
		{"/usr/share/man/man1", "ls.1", "file", "doc", 4, ".1"},
	}
	for _, c := range cases {
		g, level, _, _, sfx := classify(c.dir, c.name, c.etype)
		if g != c.group || level != c.level || sfx != c.sfx {
			t.Errorf("classify(%q) = (%q,%d,%q), want (%q,%d,%q)",
				c.name, g, level, sfx, c.group, c.level, c.sfx)
		}
	}
}

func TestNormalizeKeySortsNaturally(t *testing.T) {
	names := []string{
		"file10.txt", "file2.txt", "file1.txt", "The Matrix.mkv",
		"Matrix, The.mkv", "notes.txt",
	}
	keys := make([]string, len(names))
	for i, n := range names {
		keys[i] = normalizeKey(tclRoot(n), tclExt(n))
	}
	sort.SliceStable(keys, func(i, j int) bool { return compareDictionary(keys[i], keys[j]) < 0 })
	got := strings.Join(keys, "|")
	// numbers compare by value, "The" and "a/an" are ignored
	want := "file~1.txt|file~2.txt|file~10.txt|Matrix, The.mkv|Matrix.mkv|notes.txt"
	if got != want {
		t.Errorf("sorted keys = %q, want %q", got, want)
	}
}

func TestSeriesEq(t *testing.T) {
	// strict rule (default since 1.1.4): one run of digits, or a single index
	// letter at the end of the stem
	strict := []struct {
		a, b string
		want bool
	}{
		{"episode 01.mkv", "episode 02.mkv", true},
		{"episode 09.mkv", "episode 10.mkv", true},
		{"frame 001.dng", "frame 002.dng", true},
		{"IMG_0001.jpg", "IMG_0002.jpg", true},
		{"part a.txt", "part b.txt", true},
		{"track A.flac", "track B.flac", true},
		{"a.txt", "b.txt", true},
		// alphabetical neighbours are not a series
		{"abc.txt", "abd.txt", false},
		{"one.txt", "two.txt", false},
		// a series needs one difference, not several
		{"a-very-long-name-01.txt", "b-very-long-name-02.txt", false},
		// identical names are not a series either
		{"same.txt", "same.txt", false},
		{"short1.txt", "longer1.txt", false},
	}
	for _, c := range strict {
		if got := seriesEqStrict(c.a, c.b); got != c.want {
			t.Errorf("strict seriesEq(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}

	// the Tcl rule, available as --legacy-series: far too lenient, but that is
	// exactly what the original does
	loose := []struct {
		a, b string
		want bool
	}{
		{"episode 01.mkv", "episode 02.mkv", true},
		{"abc.txt", "abd.txt", true},
		{"a.txt", "b.txt", true},
		{"short1.txt", "longer1.txt", false},
		{"a-very-long-name-01.txt", "b-very-long-name-02.txt", false},
	}
	for _, c := range loose {
		if got := seriesEqLoose(c.a, c.b); got != c.want {
			t.Errorf("loose seriesEq(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestSeriesCollapses(t *testing.T) {
	var l []*Entry
	for _, n := range []string{"ep 01.mkv", "ep 02.mkv", "ep 03.mkv", "ep 04.mkv", "other.mkv"} {
		l = append(l, &Entry{Tail: n, Type: "file"})
	}
	out, n := series(l)
	if n != 3 || len(out) != 2 {
		t.Fatalf("collapsed=%d entries=%d, want 3 and 2", n, len(out))
	}
	if out[0].Post != "...4" {
		t.Errorf("series marker = %q, want ...4", out[0].Post)
	}
}

func TestShorten(t *testing.T) {
	c := &Ctx{opts: defaultOptions()}
	long := "a very long name of a document that goes on and on until the end.mkv"
	for _, w := range []int{8, 12, 20, 30, 45} {
		got := c.shorten(long, w)
		if n := displayLen(got); n > w {
			t.Errorf("shorten(%d) = %q (%d cols), does not fit", w, got, n)
		}
	}
	// a short name is never touched
	if got := c.shorten("short.txt", 40); got != "short.txt" {
		t.Errorf("shorten changed a short name: %q", got)
	}
	// the suffix and the number survive
	got := c.shorten("some long file name before the number 42 and after.mkv", 30)
	if !strings.Contains(got, ".mkv") || !strings.Contains(got, "42") {
		t.Errorf("shorten lost the interesting part: %q", got)
	}
}

func TestKmg(t *testing.T) {
	c := &Ctx{opts: defaultOptions()}
	cases := map[int64]string{
		0: "0B", 999: "999B", 1024: "1K", 1536: "1.5K", 3 * 1024 * 1024: "3M",
		3200000: "3.1M", 5 << 30: "5G",
	}
	for size, want := range cases {
		if got := strings.TrimSpace(c.kmg(size, "OFF")); got != want {
			t.Errorf("kmg(%d) = %q, want %q", size, got, want)
		}
	}
}
