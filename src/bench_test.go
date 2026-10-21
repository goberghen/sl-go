package main

// Measurement harness, not a test: `make bench`.
//
// The benchmarks run the real pipeline in-process -- the same argument parsing
// and the same Ctx fields as main() -- so the numbers include the directory
// content scan that a hand-built Ctx tends to forget. Output goes to
// io.Discard; the process start (about 1.4 ms, of which ~1.0 ms is fork/exec)
// is not part of them, which is why the external timings are higher.
//
// Names: Benchmark<Pipeline|Matcher><Case>. The *Regexp variants exist so that
// every hand-written matcher in match.go can be compared with the pattern it
// replaced: if a fast path ever gets slower than its regexp, the point of the
// exercise is gone.

import (
	"bufio"
	"io"
	"testing"
	"time"
)

// listReal runs one listing the way main() does.
func listReal(dir string) {
	opts, files, err := parseArgs([]string{"--no-color", "-1", dir})
	if err != nil {
		panic(err)
	}
	c := &Ctx{opts: opts, w: bufio.NewWriter(io.Discard), errw: io.Discard, now: time.Now()}
	c.width = 80
	c.colorOn = false
	if opts.Autosearch != "" {
		c.autoRe, c.autoFast = autosearchRe, opts.Autosearch == defaultAutosearch && !benchSlowAutosearch
	}
	listArgs(c, files)
	c.w.Flush()
}

func benchDir(b *testing.B, dir string) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		listReal(dir)
	}
}

// The five directories that stand for the five shapes of work: many files,
// many small directories, deep hierarchies, man pages (all suffixed names),
// and a directory of symlinks.
func BenchmarkListDoc(b *testing.B) { benchDir(b, "/usr/share/doc") }
func BenchmarkListInc(b *testing.B) { benchDir(b, "/usr/include") }
func BenchmarkListEtc(b *testing.B) { benchDir(b, "/etc") }
func BenchmarkListMan(b *testing.B) { benchDir(b, "/usr/share/man") }
func BenchmarkListBin(b *testing.B) { benchDir(b, "/usr/bin") }
func BenchmarkListLib(b *testing.B) { benchDir(b, "/usr/lib") }

// benchNames is the fixed corpus for the matcher benchmarks. It is deliberately
// boring and realistic: the shapes that appear in a package directory, with a
// wrapper suffix half the time.
var benchNames = func() []string {
	base := []string{
		"changelog.gz", "copyright", "README", "TODO", "NEWS.Debian.gz",
		"README.Debian", "examples", "changelog.Debian.gz", "foo.1.gz",
		"index.html", "main.c", "Makefile", "data.json", "song.mp3",
		"photo.jpg", "notes.txt",
	}
	out := make([]string, 0, 1200)
	for i := 0; i < 75; i++ {
		out = append(out, base...)
	}
	return out
}()

func benchMatcher(b *testing.B, f func(string)) {
	for i := 0; i < b.N; i++ {
		for _, n := range benchNames {
			f(n)
		}
	}
}

func BenchmarkMatcherNotable(b *testing.B) { benchMatcher(b, func(n string) { notableName(n) }) }
func BenchmarkMatcherNotableRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { notableRe.MatchString(n) })
}
func BenchmarkMatcherIgnorable(b *testing.B) { benchMatcher(b, func(n string) { ignorableName(n) }) }
func BenchmarkMatcherIgnorableRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { ignorableRe.MatchString(n) })
}
func BenchmarkMatcherDev(b *testing.B) { benchMatcher(b, func(n string) { devName(n) }) }
func BenchmarkMatcherDevRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { devRe.MatchString(n) })
}
func BenchmarkMatcherWrapper(b *testing.B) { benchMatcher(b, func(n string) { wrapperName(n) }) }
func BenchmarkMatcherWrapperRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { wrapperRe.MatchString(n) })
}
func BenchmarkMatcherManPage(b *testing.B) { benchMatcher(b, func(n string) { manPageName(n) }) }
func BenchmarkMatcherManPageRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { manPageRe.MatchString(n) })
}
func BenchmarkMatcherBackup(b *testing.B) { benchMatcher(b, func(n string) { autoBkupBase(n) }) }
func BenchmarkMatcherBackupRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { autoBkupRe.FindStringSubmatch(n) })
}
func BenchmarkMatcherAutosearch(b *testing.B) {
	benchMatcher(b, func(n string) { autosearchSpan(n) })
}
func BenchmarkMatcherAutosearchRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { autosearchRe.FindStringIndex(n) })
}
func BenchmarkMatcherDevDir(b *testing.B) { benchMatcher(b, func(n string) { devDirName(n) }) }
func BenchmarkMatcherDevDirRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { devDirRe.MatchString(n) })
}
func BenchmarkMatcherNormalizeKey(b *testing.B) {
	benchMatcher(b, func(n string) { normalizeKey(n, "") })
}
func BenchmarkMatcherNormalizeKeyRegexp(b *testing.B) {
	benchMatcher(b, func(n string) { normalizeKeyRegexp(n, "") })
}

// benchSlowAutosearch, when flipped, runs the layout through the original
// regular expression instead of the string matcher: mix go test -ldflags -X.
var benchSlowAutosearch = false
