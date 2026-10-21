package main

// Equivalence tests for the hand-written matchers in match.go.
//
// The rule of the port is that a faster predicate may never change which names
// are grouped or marked, so every function here is compared against the regexp
// it replaces:
//
//	synthetic corpus  edge cases, Unicode, newlines, empty strings -- always
//	-corpus           every name under /usr, /etc, /opt (opt-in, slow)

import (
	"flag"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var corpusRoots = flag.String("corpus", "", "comma separated roots to walk for the equivalence test")

// synthNames is deliberately nasty: every pattern's edge cases, both real file
// names and things that only a Unix filesystem can produce.
func synthNames() []string {
	names := []string{
		// empty and near-empty
		"", ".", "..", "/", "~", "~~", "~~~", "#", "##", "#a#", "#a", "a#", ".#", ".#a", ".a#",
		// notable
		"README", "readme", "README.md", "ReAdMe.txt", "READMEX", "xREADME",
		"TODO", "todo", "TODOS", "mytodo", "todo.txt", "TOSORT", "tosort",
		"notes", "NOTES.txt", "mynotes", "notes~",
		"Makefile", "makefile", "GNUmakefile", "Makefile.am", "myMakefile",
		"configure", "configure.ac", "configure~", "xconfigure",
		"build.xml", "BUILD.XML", "mybuild.xml.bak", "build.xml~",
		"index.html", "index.htm", "INDEX.HTML", "index.htmlx", "myindex.html.txt", "index.html~",
		// ignorable
		"foo~", "foo.~1.2.~", "#emacs#", "#emacs", "emacs#", ".#emacs", ".gitignore",
		"#a\n#", "#a\r#", "#\n#", "a~b", "foo~~",
		// dev
		"notes.c", "changelog", "ChangeLog", "Todo", "todo", "xnotes", "notes.txt",
		// wrapper
		"a.gz", "a.GZ", "a.z", "a.bz2", "a.uu", "a.hqx", "a.gz~", "a.zip", "gz", ".gz",
		"foo.tar.gz", "foo.tar.bz2", "foo.tgz",
		// backups
		"a.~1.2.~", "a.~1.~", "a.~.2.~", "a.~1.2~", "~1.2.~", ".~1.2.~",
		"x.~1.2.~3.4.~", "x.~12.345.~", "x.~1.2.~extra~", "foo.c.~1.7.~", "a~", "a~~",
		"a.~~", "a.~1.2.a~", "a.~a.b.~", "a.~1.2..~", "a..~1.2.~",
		// man pages
		"foo.1", "foo.2", "foo.9", "foo.0", "foo.3.gz", "foo.1x", "dir/foo.5", "dir/foo",
		"foo.o", "foo.l", "foo.n", "foo.m", "a.b.1", ".1", "x/.1", "x/foo.1/bar",
		"man1/foo.1", "foo.1\nbar", "foo.\n1",
		// autosearch
		"urgent", "my urgent notes", "(urgent)", "aurgent", "1urgent2", "x urgent",
		"urgent ", " urgent", "password", "passwords", "-password-", "pw", "-pw-", "pw-", "-pw",
		"live", "-live-", "_live_", "LIVE", "Live-", "x-live", "my-live-stream", "urgentpassword",
		"a_live_b", "a.live.b", "UrGeNt!",
		// Unicode
		"читай", "ЧИТАЙ", "résumé", "RÉSUMÉ", "notes—файл", "тodo", "ТОДО",
		"ＲＥＡＤＭＥ", "readmе", "ĳ", "ſ", "K", "K",
		// ordinary names
		"main.c", "main.go", "song.mp3", "photo.JPG", "x86_64-linux-gnu", "12345",
		"a b c", "a\tb", "trailing.", ".hidden", "..hidden",
	}
	// every prefix/suffix combination of a few interesting fragments
	parts := []string{"", "a", ".", "~", "#", ".~1.2.", ".~1.2.~", "readme", "todo", "index.htm", "l", "/"}
	for _, p := range parts {
		for _, q := range parts {
			names = append(names, p+q)
		}
	}
	return names
}

func TestFastMatchersMatchRegexps(t *testing.T) {
	if got := notableName("README"); !got {
		t.Fatal("sanity: README must be notable")
	}
	for _, n := range synthNames() {
		if got, want := notableName(n), notableRe.MatchString(n); got != want {
			t.Errorf("notableName(%q) = %v, regexp = %v", n, got, want)
		}
		if got, want := ignorableName(n), ignorableRe.MatchString(n); got != want {
			t.Errorf("ignorableName(%q) = %v, regexp = %v", n, got, want)
		}
		if got, want := devName(n), devRe.MatchString(n); got != want {
			t.Errorf("devName(%q) = %v, regexp = %v", n, got, want)
		}
		if got, want := wrapperName(n), wrapperRe.MatchString(n); got != want {
			t.Errorf("wrapperName(%q) = %v, regexp = %v", n, got, want)
		}
		if got, want := devDirName(n), devDirRe.MatchString(n); got != want {
			t.Errorf("devDirName(%q) = %v, regexp = %v", n, got, want)
		}
		if got, want := manPageName(n), manPageRe.MatchString(n); got != want {
			t.Errorf("manPageName(%q) = %v, regexp = %v", n, got, want)
		}

		gotName, gotOK := autoBkupBase(n)
		m := autoBkupRe.FindStringSubmatch(n)
		wantOK := m != nil
		wantName := ""
		if wantOK {
			wantName = m[1]
		}
		if gotOK != wantOK || gotName != wantName {
			t.Errorf("autoBkupBase(%q) = (%q, %v), regexp = (%q, %v)",
				n, gotName, gotOK, wantName, wantOK)
		}

		lo, hi, ok := autosearchSpan(n)
		loc := autosearchRe.FindStringIndex(n)
		if !ok {
			continue // non-ASCII: the caller falls back to the regexp
		}
		if loc == nil {
			if lo != hi {
				t.Errorf("autosearchSpan(%q) = (%d,%d), regexp = no match", n, lo, hi)
			}
		} else if lo != loc[0] || hi != loc[1] {
			t.Errorf("autosearchSpan(%q) = (%d,%d), regexp = (%d,%d)", n, lo, hi, loc[0], loc[1])
		}
	}
}

// TestFastMatchersOnCorpus runs the same comparison over real file names. It is
// opt-in because it needs a populated filesystem:
//
//	go test ./src -run TestFastMatchersOnCorpus -corpus /usr,/etc
func TestFastMatchersOnCorpus(t *testing.T) {
	if *corpusRoots == "" {
		t.Skip("set -corpus /usr,/etc to run")
	}
	checked := 0
	for _, root := range strings.Split(*corpusRoots, ",") {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable directory: skip it
			}
			name := d.Name()
			ts := name
			if wrapperName(name) {
				ts = tclRoot(name)
			}
			checked++
			if got, want := notableName(ts), notableRe.MatchString(ts); got != want {
				t.Fatalf("notableName(%q) = %v, regexp = %v", ts, got, want)
			}
			if got, want := ignorableName(ts), ignorableRe.MatchString(ts); got != want {
				t.Fatalf("ignorableName(%q) = %v, regexp = %v", ts, got, want)
			}
			if got, want := devName(ts), devRe.MatchString(ts); got != want {
				t.Fatalf("devName(%q) = %v, regexp = %v", ts, got, want)
			}
			if got, want := wrapperName(name), wrapperRe.MatchString(name); got != want {
				t.Fatalf("wrapperName(%q) = %v, regexp = %v", name, got, want)
			}
			if got, want := devDirName(ts), devDirRe.MatchString(ts); got != want {
				t.Fatalf("devDirName(%q) = %v, regexp = %v", ts, got, want)
			}
			if got, want := manPageName(ts), manPageRe.MatchString(ts); got != want {
				t.Fatalf("manPageName(%q) = %v, regexp = %v", ts, got, want)
			}
			g, gok := autoBkupBase(name)
			m := autoBkupRe.FindStringSubmatch(name)
			if gok != (m != nil) || (m != nil && g != m[1]) {
				t.Fatalf("autoBkupBase(%q) = (%q,%v), regexp = %v", name, g, gok, m)
			}
			lo, hi, ok := autosearchSpan(name)
			if ok {
				loc := autosearchRe.FindStringIndex(name)
				if loc == nil && lo != hi || loc != nil && (lo != loc[0] || hi != loc[1]) {
					t.Fatalf("autosearchSpan(%q) = (%d,%d), regexp = %v", name, lo, hi, loc)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	t.Logf("compared %d real file names against their regexps", checked)
	if checked < 10000 {
		t.Errorf("corpus too small (%d names): did the walk fail?", checked)
	}
}

// TestAutosearchOptionIsUsed guards the bug this change also fixes: the value of
// --autosearch used to be ignored, because layout.go always ran the compiled-in
// pattern.
func TestAutosearchOptionIsUsed(t *testing.T) {
	o := defaultOptions()
	o.Autosearch = `xyz`
	c := &Ctx{opts: o}
	c.autoRe = regexp.MustCompile(o.Autosearch)
	c.autoFast = o.Autosearch == defaultAutosearch
	if loc := c.autosearchLoc("a-xyz-b"); loc == nil {
		t.Error("--autosearch xyz did not match a-xyz-b")
	}
	if loc := c.autosearchLoc("a-urgent-b"); loc != nil {
		t.Error("--autosearch xyz still matched the built-in pattern")
	}
}

// TestNormalizeKeyEquivalence compares the single-pass sort key with the four
// ReplaceAll calls it replaces. A mismatch here would show up as a wrong sort
// order, which is much easier to miss than a wrong grouping, so the corpus is
// deliberately dense in articles, separators and digit runs.
func TestNormalizeKeyEquivalence(t *testing.T) {
	var cases []string
	cases = append(cases, synthNames()...)
	cases = append(cases,
		"The Matrix 2.mkv", "Matrix, The 2.mkv", "an apple", "a", "an", "the",
		"a.", "a-", "a b", "A b", "THE", "The.", "..a the b", ".the a", "#1", "#a",
		"x#1y", "x#1#2y", "a  b", "a__b", "a\tb", "a - . _ b", "a2b3c4",
		"2", "a2", "a22", "a 2", "a. 2", "a~2", "a-2", "_2", "the the the",
		"into the woods", "a the b", "and/or", "vol. 2", "Volume 3",
		"тhe matrix", "a читай", "anё", "#1читай",
	)
	// exhaustive short strings over an alphabet that hits every rule
	alphabet := []string{"a", "n", "t", "h", "e", "A", "T", "1", "2", "#", "~", ".", "-", " ", "_", "\t", "/", "\n", "z", "Ч", "é"}
	var rec func(prefix string, depth int)
	rec = func(prefix string, depth int) {
		cases = append(cases, prefix)
		if depth == 0 {
			return
		}
		for _, a := range alphabet {
			rec(prefix+a, depth-1)
		}
	}
	rec("", 3)

	for _, c := range cases {
		for _, sfx := range []string{"", ".txt"} {
			if got, want := normalizeKey(c, sfx), normalizeKeyRegexp(c, sfx); got != want {
				t.Fatalf("normalizeKey(%q, %q) = %q, regexp path = %q", c, sfx, got, want)
			}
		}
	}
	t.Logf("%d names x 2 suffixes", len(cases))
}

// TestNormalizeKeyOnCorpus is the same check over real names.
func TestNormalizeKeyOnCorpus(t *testing.T) {
	if *corpusRoots == "" {
		t.Skip("set -corpus /usr,/etc to run")
	}
	for _, root := range strings.Split(*corpusRoots, ",") {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			n := d.Name()
			if got, want := normalizeKey(n, ""), normalizeKeyRegexp(n, ""); got != want {
				t.Fatalf("normalizeKey(%q) = %q, regexp = %q", n, got, want)
			}
			return nil
		})
	}
}

func TestManDirAndVideoTS(t *testing.T) {
	for _, s := range []string{
		"/man/man1", "/man/man3p", "/man/man", "/man/man1/foo", "man/man1",
		"x/man/mania", "/usr/man/man1", "/man/man1/", "man1", "/man/man/1",
		"/usr/share/man/man1/", "/MAN/MAN1", "/usr/share/man/de/man1",
	} {
		if got, want := manDirName(s), manDirRe.MatchString(s); got != want {
			t.Errorf("manDirName(%q) = %v, regexp = %v", s, got, want)
		}
	}
	for _, s := range []string{"VIDEO_TS", "video_ts", "Video_Ts", "VIDEO_TS2",
		"xVIDEO_TS", "VIDEO_TS/", "видео_ts", "VIDEO_ТS"} {
		if got, want := videoTSName(s), videoTSRe.MatchString(s); got != want {
			t.Errorf("videoTSName(%q) = %v, regexp = %v", s, got, want)
		}
	}
}

// TestNormalizeKeyRandom throws random ASCII strings at both implementations.
// The alphabet is small and hostile so the article, separator and digit rules
// collide often.
func TestNormalizeKeyRandom(t *testing.T) {
	alpha := []byte("aAntThe#-. _\t123zZ~/\n0")
	r := uint32(12345)
	next := func(n int) int { // xorshift, no imports
		r ^= r << 13
		r ^= r >> 17
		r ^= r << 5
		return int(r % uint32(n))
	}
	for i := 0; i < 200000; i++ {
		n := next(14)
		b := make([]byte, n)
		for j := range b {
			b[j] = alpha[next(len(alpha))]
		}
		s := string(b)
		if got, want := normalizeKey(s, ""), normalizeKeyRegexp(s, ""); got != want {
			t.Fatalf("normalizeKey(%q) = %q, regexp = %q", s, got, want)
		}
	}
}
