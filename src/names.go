package main

// Names: how a file name is turned into (group, ignore level, title, suffix).
// Port of the GROUP/EQ tables and of the procs groupbyname/normalizekey/prettyname.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	wrapperRe    = regexp.MustCompile(`(?i)\.(gz|z|bz2|uu|hqx)$`)
	devRe        = regexp.MustCompile(`(?i)notes|^todo|changelog`)
	devDirRe     = regexp.MustCompile(`^lib$|^src|(^|[ _-])test|jar|dump`)
	manDirRe     = regexp.MustCompile(`/man/man[^/]+$`)
	manPageRe    = regexp.MustCompile(`^.+?\.[1-9oln][^/]*$`)
	notableRe    = regexp.MustCompile(`(?i)README|TODO$|notes|TOSORT|Makefile$|configure$|build.xml|index.html?`)
	autoBkupRe   = regexp.MustCompile(`(.*?)(\.~\d+\.\d+\.)?~$`)
	ignorableRe  = regexp.MustCompile(`~$|^#.*#$|^\.#`)
	videoTSRe    = regexp.MustCompile(`(?i)^VIDEO_TS`)
	autosearchRe = regexp.MustCompile(`(?i)\W(urgent|password|pw|live)\W`)
)

// group is one row of the GROUP table: a display order, a name and sort/display flags.
type group struct {
	name   string
	by     string // "byascii", "bysuffix", "by1" or "" for Tcl's dictionary sort
	title  bool   // strip suffix and prettify the name when -title is given
	nosfx  bool   // -nosfx may drop the suffix for this group
	prefix bool   // common-prefix elision may run in this group

	// entries are exact names or suffixes; a leading "-" means "in this group,
	// but ignored unless -a".
	entries []string
}

// The order of this slice is the display order of the groups (as in the Tcl script,
// with "dir" and "other" in the middle so that folders sit between types).
var groupDefs = []group{
	{name: "special", by: "byascii"},
	{name: "dir", by: "by1", title: true, prefix: true},
	{name: "build", by: "byascii", entries: []string{
		"Makefile", "GNUmakefile", "build.xml", "-build.number", "configure", "INSTALL", "README"}},
	{name: "app", by: "bysuffix", entries: []string{
		"-.so", "-.a", ".jar", ".exe", ".dll", "-core"}},
	{name: "dev", entries: []string{
		".c", ".cpp", ".cc", ".m", "-.h", "-.o", ".java", "-.class", ".scala", ".f",
		".tcl", ".sh", ".py", "-.pyc", ".pl", ".scpt", ".m4", ".lua",
		".nib", ".y",
		"-CVS", "-RCS", "-TAGS", "-tags", ".el", "-.elc"}},
	// Apple II disk images and their wrappers. The original list is from
	// February 2012; the entries marked "(new)" were added by the Go port:
	// WOZ flux images arrived in 2018 and are the de-facto standard for
	// copy-protected disks, while HDV/D13/EDD/NB2/N35/D16 were missing from the
	// original table. Unlike the Tcl version, this list is not switchable: a
	// retro user should not have to remember a flag to see their own diskettes.
	{name: "A2", title: true, entries: []string{
		".dsk", ".do", ".po", ".2mg", ".2img", ".shk", ".sdk", ".sea", ".bsq", ".bxy", ".bny", ".bsc", ".dc",
		".woz", ".hdv", ".d13", ".edsk", ".edd", ".nb2", ".n35", ".d16"}}, // (new)

	{name: "WWW", entries: []string{".html", ".htm", ".shtml", ".php", ".php3", ".ars"}},
	{name: "aux", by: "bysuffix", entries: []string{
		".css", ".js", ".url", ".webloc", "robots.txt", ".json"}},

	{name: "doc", prefix: true, entries: []string{
		".txt", ".man", ".1", ".3", ".dox", ".chm", ".tex", ".bib", ".sty", ".dvi",
		".texi", ".info", ".nfo", ".readme", "Index"}},
	{name: "book", title: true, entries: []string{
		".ps", ".pdf", ".djvu", ".epub", ".mobi", ".cbr", ".cbz"}},
	{name: "document", title: true, entries: []string{
		".indd", ".odt", ".doc", ".rtf", ".docx", ".ppt", ".pptx", ".xls", ".xlsx"}},
	{name: "image", title: true, nosfx: true, prefix: true, entries: []string{
		".jpg", ".jpeg", ".jp2", ".png", ".gif", ".tiff", ".tif", ".dng", ".psd",
		".xbm", ".xpm", ".pbm", ".pgm", ".ppm", ".ico", ".bmp", ".raw", "Thumbs.db"}},
	{name: "vector", title: true, entries: []string{".ai", ".svg", ".wmf", ".emf"}},
	{name: "audio", title: true, nosfx: true, prefix: true, entries: []string{
		".aac", ".mp3", ".m4a", ".m4b", ".m4p", ".aif", ".aiff", ".snd", ".flac",
		".ogg", ".oga", ".mid", ".wav", ".wma", ".ra", ".mod"}},
	{name: "video", title: true, nosfx: true, prefix: true, entries: []string{
		".mpg", ".mpeg", ".avi", ".mov", ".wmv", ".mkv", ".mp4", ".m4v", ".flv",
		".f4v", ".asf", ".ram", ".rm", ".rmvb", ".divx", ".vob", ".iso"}},
	{name: "font", title: true, nosfx: true, prefix: true, entries: []string{
		".pfb", ".pfa", "-.afm", "-.pfm", ".ttf", ".ttc", ".dfont", ".otf", ".otc",
		".woff", "-fonts.dir", "-fonts.list", "-fonts.scale", "-encodings.dir"}},

	{name: "data", entries: []string{
		".xml", ".plist", ".strings", ".data", ".dat", ".sfv", ".md5", ".bin"}},
	{name: "other", by: "byascii"},
	{name: "archive", title: true, prefix: true, entries: []string{
		".zip", ".tar", ".tgz", ".xz", ".rar", ".sit", ".sitx", ".cpt", ".dmg"}},

	{name: "dotfile"},
	{name: "Mac", entries: []string{"-.DS_Store", "-Desktop DB", "-Desktop DF"}},
	{name: "ignore", entries: []string{
		".bak", "bkup", ".bkup", ".bup", "tmp", ".tmp", "-old", ".old", "-obsolete", "ignore"}},
}

var (
	groupOrder []string
	groupBy    = map[string]*group{}
	// eq maps an exact name or a suffix to a group; a leading "-" means
	// "belongs to this group but is ignored unless -a is given".
	eq = map[string]string{}
)

func init() {
	for i := range groupDefs {
		g := &groupDefs[i]
		groupOrder = append(groupOrder, g.name)
		groupBy[g.name] = g
		for _, x := range g.entries {
			if strings.HasPrefix(x, "-") {
				// "-.h": belongs to dev, but stays hidden until -a
				eq[x[1:]] = "-" + g.name
			} else {
				eq[x] = g.name
			}
		}
	}
}

// tclExt mimics Tcl's "file extension": the suffix including the dot, except for
// leading-dot names (".bashrc" has no extension).
func tclExt(p string) string {
	base := p
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		base = p[i+1:]
	}
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		return base[i:]
	}
	return ""
}

// tclRoot mimics Tcl's "file rootname".
func tclRoot(p string) string { return p[:len(p)-len(tclExt(p))] }

// classify is groupbyname: name -> (group, ignore level, name w/o wrapper, title, suffix).
//
// Ignore levels: 4 normal, 3 forced-ignored (shown only with -a),
// 2 dotfile, 1 ignored.
func classify(dir, name, etype string) (g string, iglevel int, ts, title, sfx string) {
	ts = name
	if wrapperName(name) {
		ts = tclRoot(name)
	}
	if etype == "file" && tclRoot(ts) != "" {
		title = tclRoot(ts)
		sfx = strings.ToLower(tclExt(ts))
	} else {
		title = ts
		sfx = ""
	}

	switch {
	case eq[ts] != "":
		g = eq[ts] // exact name can't be beaten
	case ignorableName(ts):
		g = "ignore" // regexps take precedence over suffixes
	case strings.HasPrefix(ts, "."):
		g = "dotfile"
	case devName(ts) && (sfx == "" || sfx == ".txt"):
		g = "dev"
	case manDirName(dir) && manPageName(ts):
		g = "doc"
	case eq[sfx] != "":
		g = eq[sfx]
	default:
		g = "other"
	}

	// user rules win over the built-in tables; they are matched against the
	// same classification name (wrapper suffix already stripped)
	if rg, ok := ruleGroup(ts); ok {
		g = rg
	}

	switch {
	case strings.HasPrefix(g, "-"):
		g, iglevel = g[1:], 3
	case g == "dotfile":
		iglevel = 2
	case g == "ignore":
		iglevel = 1
	default:
		iglevel = 4
	}
	return g, iglevel, ts, title, sfx
}

var articleRe = regexp.MustCompile(`(?i)(^|\W)(a|an|the)\W`)

// The four patterns normalizeKeyRegexp applies, kept for the reference path in
// match.go (they are also what normalizeKey's single pass reproduces).
var (
	hashNumRe  = regexp.MustCompile(`#(\d)`)
	numAfterRe = regexp.MustCompile(`(\D)(\d+)`)
	sepRe      = regexp.MustCompile(`[-. _\t]+`)
)

// prettyName: camelCase -> "camel Case", separators -> space (only with -title).
func prettyName(d string) string {
	d = camelRe.ReplaceAllString(d, "${1} ${2}")
	return sepRe2.ReplaceAllString(d, " ")
}

var (
	camelRe = regexp.MustCompile(`([a-z])([A-Z])`)
	sepRe2  = regexp.MustCompile(`[._ ]+`)
)

// compareDictionary is Tcl's lsort -dictionary: case-insensitive, numbers compare
// by value, and a leading run of non-alphanumerics is ignored.
func compareDictionary(a, b string) int {
	ia, ib := 0, 0
	// skip leading non-alphanumeric clutter
	for ia < len(a) && !isAlnum(rune(a[ia])) {
		ia++
	}
	for ib < len(b) && !isAlnum(rune(b[ib])) {
		ib++
	}
	for ia < len(a) && ib < len(b) {
		ca, cb := a[ia], b[ib]
		// numeric chunk?
		if isDigit(ca) && isDigit(cb) {
			na, la := readNumber(a[ia:])
			nb, lb := readNumber(b[ib:])
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			ia += la
			ib += lb
			continue
		}
		ra, rb := unicode.ToLower(rune(ca)), unicode.ToLower(rune(cb))
		if ra != rb {
			if ra < rb {
				return -1
			}
			return 1
		}
		ia++
		ib++
	}
	switch {
	case ia < len(a):
		return 1
	case ib < len(b):
		return -1
	}
	return 0
}

func readNumber(s string) (int64, int) {
	i := 0
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	n, err := strconv.ParseInt(s[:i], 10, 64)
	if err != nil {
		n = 0
	}
	return n, i
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlnum(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
func compareASCII(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// ------------------------------------------------------------- user rules
//
// The Tcl original let you redefine classify() in ~/.sl.tcl, i.e. run arbitrary
// code. A static binary cannot host an interpreter, so the same job is done
// with one declarative line per rule:
//
//	rule = <regexp> -> <group>
//
// The regexp is Go's syntax, unanchored and case-sensitive unless it says
// "(?i)". It is matched against the classification name: the entry's name with
// a compression wrapper stripped, so "foo.1.gz" is matched as "foo.1" -- the
// same string every other name test in this file sees.
//
// Rules are tried in order and the first match wins; a rule overrides the
// built-in tables (suffixes, exact names, README/Makefile detection), but not
// the "ignore" semantics: forcing the group "ignore" hides the entry the same
// way an ignorable name is hidden, which is what people used the Tcl hooks for
// ("hide everything ending in .tmp").

type userRule struct {
	re    *regexp.Regexp
	group string
}

var userRules []userRule

// compileRules parses and validates the rules once, at startup, so that a typo
// is an error and not a surprise in the middle of a listing.
func compileRules(specs []string) error {
	userRules = nil
	for _, raw := range specs {
		spec := strings.TrimSpace(raw)
		if spec == "" {
			continue
		}
		i := strings.LastIndex(spec, "->")
		if i < 0 {
			return fmt.Errorf("rule %q: expected \"regexp -> group\"", raw)
		}
		pattern := strings.TrimSpace(spec[:i])
		group := strings.TrimSpace(spec[i+2:])
		if pattern == "" {
			return fmt.Errorf("rule %q: empty pattern", raw)
		}
		if groupBy[group] == nil {
			return fmt.Errorf("rule %q: unknown group %q (known: %s)",
				raw, group, strings.Join(groupNames(), " "))
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("rule %q: %v", raw, err)
		}
		userRules = append(userRules, userRule{re: re, group: group})
	}
	return nil
}

// groupNames lists the groups a rule may name: the table order first, then the
// names that exist without a table entry. Duplicates are dropped, so the list
// can be printed verbatim in an error message or the manual.
func groupNames() []string {
	out := make([]string, 0, len(groupDefs)+4)
	seen := map[string]bool{}
	for _, name := range append(groupNamesOf(groupDefs), "file", "other", "ignore", "dotfile") {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

func groupNamesOf(defs []group) []string {
	out := make([]string, 0, len(defs))
	for _, g := range defs {
		out = append(out, g.name)
	}
	return out
}

// ruleGroup returns the group forced by the first matching rule.
func ruleGroup(ts string) (string, bool) {
	for _, r := range userRules {
		if r.re.MatchString(ts) {
			return r.group, true
		}
	}
	return "", false
}
