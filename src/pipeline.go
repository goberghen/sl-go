package main

// The pipeline: collect -> mark up -> order -> lay out -> print.
// This is sl2() plus main() from the Tcl script.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Ctx struct {
	opts    *Options
	w       *bufio.Writer
	errw    io.Writer
	width   int
	colorOn bool
	now     time.Time
	logs    []string

	annotateRe *regexp.Regexp    // --annotate-match
	batchAnn   map[string]string // path -> annotation, filled once per directory
	home       string            // absolute path of $HOME

	autoRe   *regexp.Regexp // --autosearch, compiled once
	autoFast bool           // autoRe is the built-in pattern, so the fast path is valid
}

func (c *Ctx) logf(format string, args ...any) {
	if c.opts.Log {
		c.logs = append(c.logs, fmt.Sprintf(format, args...))
	}
}

// list prints one directory: either the whole directory (paths == nil) or the
// named files that live in it.
func (c *Ctx) list(dir string, paths []string, level int) {
	if paths == nil {
		names, err := readNames(dir)
		if err != nil {
			fmt.Fprintf(c.errw, "sl: %s: %v\n", dir, err)
			return
		}
		if len(names) == 0 {
			return
		}
		paths = make([]string, len(names))
		for i, n := range names {
			if dir == "" || dir == "." {
				paths[i] = n
			} else {
				paths[i] = filepath.Join(dir, n)
			}
		}
	}

	// one hook run for the whole directory (--annotate-batch); it has to
	// happen before the entries are built, because the answer is attached to
	// the entry as it is made
	c.annotateBatch(dir, paths)

	listed := map[string]bool{}
	for _, p := range paths {
		listed[filepath.Base(p)] = true
	}

	cnt := &counters{}
	var igl []string
	var list []*Entry
	for _, p := range paths {
		name := filepath.Base(p)
		if e := c.buildEntry(dir, name, p, listed, cnt, &igl); e != nil {
			list = append(list, e)
		}
	}
	if len(igl) > 0 {
		c.logf("ignored (%s)", strings.Join(igl, " "))
	}
	c.logf("collect %d", len(paths))
	maxlen := 0
	for _, e := range list {
		if l := displayLen(e.Tail); l > maxlen {
			maxlen = l
		}
	}
	width := c.width - 1
	cols := width / (maxlen + 1 + 1)
	if cols == 0 {
		cols = 1
	}
	c.logf("cf ls -B %d/(%d+1+1) for %d = %dx%d", width, maxlen, len(list), cols, iceil(len(list), cols))

	// 2. mark up
	c.logf("mark %d", len(list))
	list = c.relative(list)
	if c.opts.Only {
		kept := list[:0]
		for _, e := range list {
			if distinctive(e) {
				kept = append(kept, e)
			}
		}
		list = kept
	}
	if len(list) == 0 {
		return
	}

	// 3a. order
	n := len(list) // datalen: counted before series/prefix collapsing
	list, gtitcnt := c.order(list)

	// 3b. lay out and print
	c.logf("display %d", len(list))
	c.render(list, gtitcnt)

	// summary line
	if c.opts.Summary > 0 && n >= c.opts.Summary {
		var b strings.Builder
		b.WriteString("  " + plural(cnt.files, "file", "files", ""))
		if cnt.files > 0 {
			b.WriteString(" (" + c.kmg(cnt.totalSize, "OFF") + ")")
		}
		if cnt.dots > 0 {
			b.WriteString(" + " + plural(cnt.dots, "dotfile", "dotfiles", c.color(".", "warning", "OFF")))
		}
		if cnt.ignored > 0 {
			b.WriteString(" + " + plural(cnt.ignored, "ignored", "ignored", ""))
		}
		if cnt.dirs > 0 {
			b.WriteString(", " + plural(cnt.dirs, "directory", "directories", ""))
			b.WriteString(" (" + plural(int(cnt.totalDir), "file", "files", "") + ")")
		}
		c.w.WriteString(b.String() + "\n")
	}

	if c.opts.Log && len(c.logs) > 0 {
		c.w.WriteString("log: " + strings.Join(c.logs, ", ") + "\n")
	}

	// autoexpand: a directory holding a single folder that holds something
	if c.opts.Expando > 0 && cnt.files <= c.opts.Expando && cnt.dirs == 1 && level < 10 {
		for _, e := range list {
			if e.isDir() && e.Size > 0 {
				sub := filepath.Join(dir, e.Tail)
				c.w.WriteString(fmt.Sprintf("\nautoexpand %s:\n", c.color(e.Tail+"/", "dir", "OFF")))
				c.w.Flush()
				c.list(sub, nil, level+1)
				break
			}
		}
	}
}

// distinctive is the -only filter: keep entries that carry some signal.
func distinctive(e *Entry) bool {
	switch e.Type {
	case "file":
		w := e.Warning
		if e.Style != styleMap["file"] || e.Pre != "  " ||
			(w != "" && w != "*" && w != "*<-") || e.Post != "" {
			return true
		}
	case "directory":
		w := e.Warning
		if e.Style != styleMap["dir"] || e.Pre != "  " ||
			(w != "" && w != "1") || displayLen(e.Post) > 1 {
			return true
		}
	}
	return false
}

func titleEntry(name string) *Entry {
	return &Entry{
		Type: "GROUP", Style: styleMap["group"], Pre: "  ", Display: name,
	}
}

// order groups, sorts inside groups, collapses series and elided prefixes.
func (c *Ctx) order(list []*Entry) ([]*Entry, int) {
	ga := map[string][]*Entry{}
	for _, e := range list {
		ga[e.Group] = append(ga[e.Group], e)
	}

	var out []*Entry
	lastgroup := "dir" // note: the first "dir" group gets no title, like the original
	gtitcnt := 0
	statsSeries, statsPfx, statsPfxCh := 0, 0, 0

	for _, gname := range groupOrder {
		gl := ga[gname]
		if len(gl) == 0 {
			continue
		}
		g := groupBy[gname]

		// dictionary sort by sort key, then drop duplicates (lsort -unique)
		sort.SliceStable(gl, func(i, j int) bool {
			return compareDictionary(gl[i].Sortkey, gl[j].Sortkey) < 0
		})
		gl = uniqueBySortkey(gl)

		switch g.by {
		case "bysuffix":
			sort.SliceStable(gl, func(i, j int) bool {
				return compareASCII(gl[i].Sfx, gl[j].Sfx) < 0
			})
		case "byascii":
			sort.SliceStable(gl, func(i, j int) bool {
				return compareASCII(gl[i].Tail, gl[j].Tail) < 0
			})
		case "by1":
			sort.SliceStable(gl, func(i, j int) bool {
				return compareASCII(firstRune(gl[i].Tail), firstRune(gl[j].Tail)) < 0
			})
		}

		if c.opts.Series > 0 && len(gl) >= c.opts.Series {
			var n int
			gl, n = seriesWith(gl, c.opts.LegacySeries)
			statsSeries += n
		}
		if c.opts.Prefix > 0 && g.prefix {
			var n1, n2 int
			gl, n1, n2 = prefixElide(gl, c.opts.Prefix)
			statsPfx, statsPfxCh = statsPfx+n1, statsPfxCh+n2
		}

		if c.opts.GroupTitle && gname != lastgroup {
			out = append(out, titleEntry(gname))
			lastgroup = gname
			gtitcnt++
		}
		out = append(out, gl...)
	}

	if statsSeries > 0 {
		c.logf("series -%d", statsSeries)
	}
	if statsPfx > 0 {
		c.logf("prefix %d (%d chars)", statsPfx, statsPfxCh)
	}
	if !c.opts.Group {
		sort.SliceStable(out, func(i, j int) bool {
			return compareDictionary(out[i].Sortkey, out[j].Sortkey) < 0
		})
	}
	return out, gtitcnt
}

func uniqueBySortkey(list []*Entry) []*Entry {
	out := list[:0]
	for i, e := range list {
		if i > 0 && e.Sortkey == list[i-1].Sortkey {
			continue
		}
		out = append(out, e)
	}
	return out
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

// listArgs is main(): split the arguments into directories to list and files to
// show, and print them with "dir:" headers when there is more than one target.
func listArgs(c *Ctx, args []string) {
	if len(args) == 0 {
		args = []string{"."}
	}

	filesByDir := map[string][]string{}
	var dirs []string
	for _, a := range args {
		fi, err := os.Stat(a)
		if err != nil {
			fmt.Fprintf(c.errw, "%s: no such file or directory\n", a)
			continue
		}
		if !isReadable(a) {
			fmt.Fprintf(c.errw, "%s: not readable\n", a)
			continue
		}
		if fi.IsDir() {
			dirs = append(dirs, a)
		} else {
			d := filepath.Dir(a)
			filesByDir[d] = append(filesByDir[d], a)
		}
	}

	fdirs := make([]string, 0, len(filesByDir))
	for d := range filesByDir {
		fdirs = append(fdirs, d)
	}
	sort.Strings(fdirs)

	total := len(fdirs) + len(dirs)
	first := true
	for _, d := range fdirs {
		if !first {
			c.w.WriteString("\n")
		}
		if total > 1 {
			c.w.WriteString(d + ":\n")
		}
		c.list(d, filesByDir[d], 0)
		first = false
	}
	for _, d := range dirs {
		if !first {
			c.w.WriteString("\n")
		}
		if total > 1 {
			c.w.WriteString(d + ":\n")
		}
		c.list(d, nil, 0)
		first = false
	}
}
