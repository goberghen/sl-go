package main

import "strings"

// Byte-level fast paths for the hot regular expressions.
//
// Where the time goes (pprof on /usr/share/doc, 55% of the run inside regexp
// backtracking), microseconds per name:
//
//	notableRe   (?i)README|TODO$|notes|TOSORT|Makefile$|configure$|build.xml|index.html?   1.42
//	autoBkupRe  (.*?)(\.~\d+\.\d+\.)?~$                                                     0.97
//	ignorableRe ~$|^#.*#$|^\.#                                                            0.64
//	devRe       (?i)notes|^todo|changelog                                                 0.53
//	autosearchRe (?i)\W(urgent|password|pw|live)\W                                       0.33
//	manPageRe   ^.+?\.[1-9oln][^/]*$                                                      0.30
//	wrapperRe   (?i)\.(gz|z|bz2|uu|hqx)$                                                  0.13
//
// Every function here is a hand-written matcher for exactly one of those fixed
// patterns. Each one is a byte loop over ASCII input and delegates to the real
// regexp for anything else, so behaviour for non-ASCII names (and for the one
// pattern where '.' cannot cross a newline) is byte-for-byte what it was.
//
// The equivalence is not assumed: match_test.go checks every function against
// its regexp on a synthetic corpus, and with -corpus on every name under /usr.

// isASCII reports whether s is pure ASCII, which is the precondition for the
// byte loops below.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// lowerB lowercases an ASCII byte.
func lowerB(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// containsFold reports whether s contains sub, ASCII case-insensitively. sub
// must already be lower case.
func containsFold(s, sub string) bool {
	n := len(sub)
	if n == 0 {
		return true
	}
	last := len(s) - n
	for i := 0; i <= last; i++ {
		j := 0
		for j < n && lowerB(s[i+j]) == sub[j] {
			j++
		}
		if j == n {
			return true
		}
	}
	return false
}

// hasPrefixFold reports whether s starts with sub, ASCII case-insensitively.
func hasPrefixFold(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i < len(sub); i++ {
		if lowerB(s[i]) != sub[i] {
			return false
		}
	}
	return true
}

// hasSuffixFold reports whether s ends with sub, ASCII case-insensitively.
func hasSuffixFold(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	off := len(s) - len(sub)
	for i := 0; i < len(sub); i++ {
		if lowerB(s[off+i]) != sub[i] {
			return false
		}
	}
	return true
}

func isDigitB(b byte) bool { return b >= '0' && b <= '9' }

func isWordByte(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '_'
}

// ---------------------------------------------------------------- notable

// notableName replaces
//
//	(?i)README|TODO$|notes|TOSORT|Makefile$|configure$|build.xml|index.html?
//
// Four of the eight alternatives look anywhere in the name (README, notes,
// TOSORT, build.xml, index.htm), three are suffix-anchored, and the `l?` on
// index.html makes it "contains index.htm".
func notableName(s string) bool {
	if !isASCII(s) {
		return notableRe.MatchString(s)
	}
	return containsFold(s, "readme") ||
		hasSuffixFold(s, "todo") ||
		containsFold(s, "notes") ||
		containsFold(s, "tosort") ||
		hasSuffixFold(s, "makefile") ||
		hasSuffixFold(s, "configure") ||
		containsFold(s, "build.xml") ||
		containsFold(s, "index.htm")
}

// ---------------------------------------------------------------- ignorable

// ignorableName replaces "~$|^#.*#$|^\.#".
//
// The newline is the only trap: '.' does not match '\n' in Go's regexp, so
// "#a\n#" is not ignorable even though s[0] and s[len-1] are '#'. A name with a
// newline takes the regexp path.
func ignorableName(s string) bool {
	if !isASCII(s) || indexByte(s, '\n') >= 0 {
		return ignorableRe.MatchString(s)
	}
	if len(s) > 0 && s[len(s)-1] == '~' {
		return true
	}
	if len(s) >= 2 && s[0] == '#' && s[len(s)-1] == '#' {
		return true
	}
	return len(s) >= 2 && s[0] == '.' && s[1] == '#'
}

// indexByte is strings.IndexByte without the dependency in this file.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------- dev

// devName replaces "(?i)notes|^todo|changelog".
func devName(s string) bool {
	if !isASCII(s) {
		return devRe.MatchString(s)
	}
	return containsFold(s, "notes") || hasPrefixFold(s, "todo") || containsFold(s, "changelog")
}

// ---------------------------------------------------------------- wrapper

// wrapperName replaces "(?i)\.(gz|z|bz2|uu|hqx)$".
func wrapperName(s string) bool {
	if !isASCII(s) {
		return wrapperRe.MatchString(s)
	}
	return hasSuffixFold(s, ".gz") || hasSuffixFold(s, ".z") || hasSuffixFold(s, ".bz2") ||
		hasSuffixFold(s, ".uu") || hasSuffixFold(s, ".hqx")
}

// ---------------------------------------------------------------- backup

// autoBkupBase replaces autoBkupRe.FindStringSubmatch(name)[1] for
//
//	(.*?)(\.~\d+\.\d+\.)?~$
//
// The trailing "~" is mandatory. The optional group is tried first (it is
// greedy), and it must end right before that "~", so the only two candidates
// are "everything before the last .~N.M.~" and "everything before the final ~".
func autoBkupBase(name string) (string, bool) {
	if !isASCII(name) {
		if m := autoBkupRe.FindStringSubmatch(name); m != nil {
			return m[1], true
		}
		return "", false
	}
	if len(name) == 0 || name[len(name)-1] != '~' {
		return "", false
	}
	core := name[:len(name)-1]
	if base, ok := stripTrailingTrace(core); ok {
		return base, true
	}
	return core, true
}

// stripTrailingTrace removes one trailing ".~<digits>.<digits>." (the middle
// part of an editor backup name such as "foo.c.~1.7.~").
func stripTrailingTrace(s string) (string, bool) {
	if len(s) < 6 || s[len(s)-1] != '.' { // ".~1.1." is the shortest
		return "", false
	}
	i := len(s) - 1 // position of the final '.'
	j := i
	for j > 0 && isDigitB(s[j-1]) {
		j--
	}
	if j == i || j == 0 || s[j-1] != '.' {
		return "", false
	}
	k := j - 1
	for k > 0 && isDigitB(s[k-1]) {
		k--
	}
	if k == j-1 || k < 2 || s[k-1] != '~' || s[k-2] != '.' {
		return "", false
	}
	return s[:k-2], true
}

// ---------------------------------------------------------------- man page

// manPageName replaces "^.+?\.[1-9oln][^/]*$": a dot, at least one character
// before it, a manual section character, and no '/' after the dot.
func manPageName(s string) bool {
	if !isASCII(s) {
		return manPageRe.MatchString(s)
	}
	lo := 1 // ".+?" needs one character before the dot
	if sl := lastIndexByte(s, '/'); sl+1 > lo {
		lo = sl + 1
	}
	for i := lo; i+1 < len(s); i++ {
		if s[i] != '.' {
			continue
		}
		c := s[i+1]
		if c >= '1' && c <= '9' || c == 'o' || c == 'l' || c == 'n' {
			return true
		}
	}
	return false
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------- autosearch

// defaultAutosearch is the built-in pattern; the fast path below is only valid
// for exactly this string, so the caller compares before calling.
const defaultAutosearch = `(?i)\W(urgent|password|pw|live)\W`

// autosearchSpan returns the byte span of the first match of
// defaultAutosearch in s. ok is false when the name has to take the regexp
// path (non-ASCII, where \W applies to whole runes).
//
// Leftmost-first, like Go's regexp: the earliest start wins, and at that start
// the alternatives are tried in their written order (urgent, password, pw,
// live).
func autosearchSpan(s string) (lo, hi int, ok bool) {
	if !isASCII(s) {
		return 0, 0, false
	}
	for i := 0; i+3 <= len(s); i++ {
		if isWordByte(s[i]) {
			continue
		}
		rest := s[i+1:]
		for _, alt := range [...]string{"urgent", "password", "pw", "live"} {
			if len(rest) <= len(alt) || !hasPrefixFold(rest, alt) {
				continue
			}
			if !isWordByte(rest[len(alt)]) {
				return i, i + 1 + len(alt) + 1, true
			}
		}
	}
	return 0, 0, true // scanned everything, no match
}

// autosearchLoc applies the configured autosearch pattern to one printed cell.
//
// Until now the option was only an on/off switch: layout.go always ran the
// compiled-in default and ignored the value given with --autosearch. The value
// is now compiled (and rejected if it is not a valid regexp), and the byte loop
// above is used only when the pattern really is the built-in one.
func (c *Ctx) autosearchLoc(s string) []int {
	if c.autoRe == nil {
		return nil
	}
	if c.autoFast {
		if lo, hi, ok := autosearchSpan(s); ok {
			if lo == hi {
				return nil
			}
			return []int{lo, hi}
		}
	}
	return c.autoRe.FindStringIndex(s)
}

// --------------------------------------------------- man dirs and VIDEO_TS

// manDirName replaces "/man/man[^/]+$": the path names a man section directory.
func manDirName(p string) bool {
	if !isASCII(p) {
		return manDirRe.MatchString(p)
	}
	// the last component must be "man" + at least one non-slash character,
	// and must sit directly under a directory called "man"
	slash := strings.LastIndexByte(p, '/')
	if slash < 0 {
		return false
	}
	comp := p[slash+1:]
	if len(comp) < 4 || comp[:3] != "man" {
		return false
	}
	return strings.HasSuffix(p[:slash], "/man")
}

// videoTSName replaces "(?i)^VIDEO_TS": a video folder marker.
func videoTSName(s string) bool {
	if !isASCII(s) {
		return videoTSRe.MatchString(s)
	}
	return hasPrefixFold(s, "video_ts")
}

// ------------------------------------------------------------- normalizeKey

// normalizeKeyRegexp is the original implementation, kept as the reference the
// fast path is tested against (and as the fallback for non-ASCII names).
func normalizeKeyRegexp(t, sfx string) string {
	s := t
	pfx := ""
	if strings.HasPrefix(s, ".") {
		pfx = "."
		s = s[1:]
	}
	s = hashNumRe.ReplaceAllString(s, "${1}")
	s = numAfterRe.ReplaceAllString(s, "${1}~${2}")
	s = sepRe.ReplaceAllString(s, " ")
	s = articleRe.ReplaceAllString(s, "${1}")
	if s == "" {
		return t
	}
	return pfx + s + sfx
}

// normalizeKey builds the dictionary sort key in a single pass: the four
// ReplaceAll calls of the original cost 11% of a /usr/share/doc run.
//
//	#(\d)          -> \1          drop the hash before a number
//	(\D)(\d+)      -> \1~\2       "x2" sorts like "x 2"
//	[-. _\t]+      -> " "         separators collapse to one space
//	(^|\W)(a|an|the)\W -> \1      drop a leading article
//
// Each rule is applied to the result of the previous one, exactly as before,
// but all four run over the same byte buffer. The article rule keeps the
// regexp's alternation order (a, then an, then the) and its left-to-right scan.
// Non-ASCII input, where \W and case folding are rune-shaped, goes to the
// regexp path.
func normalizeKey(t, sfx string) string {
	if !isASCII(t) {
		return normalizeKeyRegexp(t, sfx)
	}
	s := t
	pfx := ""
	if len(s) > 0 && s[0] == '.' {
		pfx = "."
		s = s[1:]
	}

	// rules 1-3 in one pass over s
	out := make([]byte, 0, len(s)+8)
	for i := 0; i < len(s); {
		c := s[i]
		if c == '#' && i+1 < len(s) && isDigitB(s[i+1]) {
			i++ // rule 1: drop the hash
			continue
		}
		if isDigitB(c) {
			// rule 2: prefix a digit run with '~' unless it starts the string
			// or directly follows a digit (the regexp needs a preceding \D)
			if len(out) > 0 && !isDigitB(out[len(out)-1]) {
				out = append(out, '~')
			}
			for i < len(s) && isDigitB(s[i]) {
				out = append(out, s[i])
				i++
			}
			continue
		}
		if c == '-' || c == '.' || c == ' ' || c == '_' || c == '\t' {
			// rule 3: a run collapses to one space, even at the start
			if len(out) == 0 || out[len(out)-1] != ' ' {
				out = append(out, ' ')
			}
			i++
			continue
		}
		out = append(out, c)
		i++
	}

	// rule 4: (^|\W)(a|an|the)\W -> \1
	//
	// The scan is leftmost-first: at each position the empty ^ branch is tried
	// first, then the \W branch, which consumes the character in front of the
	// article. That is why "a the b" loses only the first "a": after the match
	// the scan resumes past the consumed space, where \W can no longer match.
	res := make([]byte, 0, len(out))
	kept := 0
	for k := 0; k < len(out); k++ {
		var start int // where the article would begin, -1 for no candidate
		if k == 0 {
			start = 0
		} else {
			start = -1
		}
		for {
			if start >= 0 {
				if n := articleLen(out, start); n > 0 {
					// group 1 is kept: the character matched by \W, or
					// nothing for the ^ alternative
					grp := 0
					if start > k || k > 0 {
						grp = 1
					}
					res = append(res, out[kept:k+grp]...)
					kept = start + n + 1 // n letters plus the trailing \W
					k = kept - 1
					break
				}
			}
			// the \W alternative: out[k] is the character \W matches
			if start != k+1 && !isWordByte(out[k]) {
				start = k + 1
				continue
			}
			break
		}
	}
	res = append(res, out[kept:]...)

	if len(res) == 0 {
		return t
	}
	if pfx == "." {
		return "." + string(res) + sfx
	}
	return string(res) + sfx
}

// articleLen reports how many bytes of the article (a|an|the) start at i,
// provided a \W follows it. Zero means "no match here". The order of the
// alternatives is the regexp's: a, then an, then the.
func articleLen(b []byte, i int) int {
	if i >= len(b) {
		return 0
	}
	switch lowerB(b[i]) {
	case 'a':
		if i+2 < len(b) && lowerB(b[i+1]) == 'n' && !isWordByte(b[i+2]) {
			return 2
		}
		if i+1 < len(b) && !isWordByte(b[i+1]) {
			return 1
		}
	case 't':
		if i+3 < len(b) && lowerB(b[i+1]) == 'h' && lowerB(b[i+2]) == 'e' && !isWordByte(b[i+3]) {
			return 3
		}
	}
	return 0
}

// ------------------------------------------------------------- devDirName

// devDirName replaces "^lib$|^src|(^|[ _-])test|jar|dump": the directory-name
// hint that forces the "dev" group. No case folding and no dot, so a byte scan
// is exact for any input, including UTF-8.
func devDirName(s string) bool {
	if s == "lib" || strings.HasPrefix(s, "src") {
		return true
	}
	if strings.Contains(s, "jar") || strings.Contains(s, "dump") {
		return true
	}
	if strings.HasPrefix(s, "test") {
		return true
	}
	for i := 1; i < len(s); i++ {
		switch s[i-1] {
		case ' ', '_', '-':
			if strings.HasPrefix(s[i:], "test") {
				return true
			}
		}
	}
	return false
}
