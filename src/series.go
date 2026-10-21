package main

// Series collapsing and common-prefix elision.
// Port of proc series / serieseq / prefix.

import (
	"regexp"
	"strings"
	"unicode"
)

var relzRe = regexp.MustCompile(` [.\d]{1,4}[BKMGTPEZY]`)

// series turns runs of three or more "the same name with a counter" into the
// first entry followed by "...N", so a season of episodes becomes one line.
func series(list []*Entry) ([]*Entry, int) {
	return seriesWith(list, false)
}

// seriesWith collapses runs; loose selects the Tcl rule for "same series".
func seriesWith(list []*Entry, loose bool) ([]*Entry, int) {
	eq := seriesEqStrict
	if loose {
		eq = seriesEqLoose
	}
	out := make([]*Entry, 0, len(list))
	cnt := 0

	for i := 0; i < len(list); {
		var t *Entry
		relz := 0
		j := i
		for ; j < len(list); j++ {
			e := list[j]
			if e.isDir() { // a directory is already a kind of series
				break
			}
			if j > i {
				if !eq(t.Tail, e.Tail) || e.Warning != "" || !postOK(e.Post, &relz) {
					break
				}
			}
			t = e
		}

		slen := j - i
		if slen >= 3 {
			first := list[i].clone()
			post := first.Post
			if relz > 0 && !relzRe.MatchString(post) {
				post += " " + itoa(first.Size) // size of the first as representative
			}
			first.Post = post + "..." + itoa(int64(slen))
			out = append(out, first)
			cnt += slen - 1
			i = j
		} else {
			out = append(out, list[i])
			i++
		}
	}
	return out, cnt
}

// postOK reports whether a postfix is "empty or just a size", which is what may
// be swallowed by a series.
func postOK(post string, relz *int) bool {
	if post == "" {
		return true
	}
	m := relzRe.FindString(post)
	if m != "" && len(post) == len(m) {
		*relz++
		return true
	}
	return false
}

// seriesEq reports whether two names are "the same name with a counter".
//
// Two rules are supported:
//
//	strict (default)  the names are identical except for ONE run of digits,
//	                  e.g. "episode 01.mkv" / "episode 02.mkv". A single
//	                  differing LETTER counts only when it is the last letter
//	                  of the stem, i.e. an index letter ("part a.txt" /
//	                  "part b.txt"), not an ordinary alphabetical neighbour
//	                  ("abc.txt" / "abd.txt").
//
//	loose  (--legacy-series)  the Tcl rule: same length, and every differing
//	                  character is a digit or a single run of letters.
//
// seriesEqStrict is the improved rule used by default since 1.1.4.
func seriesEqStrict(f1, f2 string) bool {
	a := []rune(f1)
	b := []rune(f2)
	if len(a) != len(b) {
		return false
	}
	// find the single differing run [i,j)
	i := 0
	for i < len(a) && a[i] == b[i] {
		i++
	}
	if i == len(a) {
		return false // identical: not a series, just one name twice
	}
	j := i
	for j < len(a) && a[j] != b[j] {
		j++
	}
	for k := j; k < len(a); k++ {
		if a[k] != b[k] {
			return false // a second, separate difference
		}
	}

	numeric := true
	alpha := true
	for k := i; k < j; k++ {
		if !unicode.IsDigit(a[k]) || !unicode.IsDigit(b[k]) {
			numeric = false
		}
		if !unicode.IsLetter(a[k]) || !unicode.IsLetter(b[k]) {
			alpha = false
		}
	}
	if numeric {
		return true
	}
	if alpha && j-i == 1 {
		// An index letter is a single letter that ends the stem and starts it
		// or follows a separator: "part a", "track A", but not "abc"/"abd",
		// which is just the alphabet.
		if end := stemEnd(a); j != end {
			return false
		}
		if i > 0 && (unicode.IsLetter(a[i-1]) || unicode.IsDigit(a[i-1])) {
			return false
		}
		return true
	}
	return false
}

// stemEnd is the index of the last dot that starts a suffix ("x.txt" -> 3),
// or len(r) when the name has no suffix.
func stemEnd(r []rune) int {
	for k := len(r) - 1; k > 0; k-- {
		if r[k] == '.' {
			return k
		}
	}
	return len(r)
}

// seriesEqLoose is the Tcl original's rule, kept for --legacy-series: same
// length, and the differences are digits or one run of letters.
func seriesEqLoose(f1, f2 string) bool {
	a := []rune(f1)
	b := []rune(f2)
	if len(a) != len(b) {
		return false
	}
	i := 0
	if len(a) >= 10 { // a long name must share a real prefix
		for ; i < 4; i++ {
			if a[i] != b[i] {
				return false
			}
		}
	}
	alphaseries := false
	for ; i < len(a); i++ {
		c1, c2 := a[i], b[i]
		switch {
		case c1 == c2:
		case unicode.IsDigit(c1) && unicode.IsDigit(c2):
		case alphaseries:
			return false
		case unicode.IsLetter(c1) && unicode.IsLetter(c2):
			alphaseries = true
		case unicode.IsDigit(c1) && unicode.IsLetter(c2): // 0..9 continues into a..z
			alphaseries = true
		default:
			return false
		}
	}
	return true
}

// prefixElide replaces the common prefix of a name with the same number of
// spaces, so a column of "myproject-notes", "myproject-todo" becomes
// "          notes", "          todo".
func prefixElide(list []*Entry, minPfx int) ([]*Entry, int, int) {
	out := make([]*Entry, 0, len(list))
	last := []rune("")
	cnt, cntch := 0, 0

	for _, e0 := range list {
		e := e0.clone()
		d := []rune(e.Display)
		pfx := 0
		lastc := rune(0)
		i := 0
		imax := len(last)
		if len(d) < imax {
			imax = len(d)
		}
		for ; i < imax; i++ {
			c1, c2 := last[i], d[i]
			switch {
			case c2 == ' ':
				pfx = i + 1
			case unicode.IsPunct(c2) || c2 == '\x1b':
				pfx = i
			case lastc == 0:
			case unicode.IsLower(lastc) && unicode.IsUpper(c2): // camelCase
				pfx = i
			case unicode.IsLetter(lastc) && unicode.IsDigit(c2): // letter -> digit
				pfx = i
			}
			if c1 != c2 {
				break
			}
			lastc = c2
		}
		if i == imax && !unicode.IsDigit(lastc) {
			pfx = i
		}
		if pfx >= minPfx && pfx < len(d) {
			// pad by the *width* of the elided runes, not by their count:
			// otherwise CJK names lose the column alignment
			e.Display = strings.Repeat(" ", runeSliceWidth(e.Display, pfx)) + string(d[pfx:])
			cnt++
			cntch += pfx
		}
		out = append(out, e)
		last = d
	}
	return out, cnt, cntch
}
