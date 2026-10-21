package main

// Scan: read a directory (or a list of files) and build one Entry per visible
// name. This is the "1. build up data record for each file" phase of the original.

import (
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type counters struct {
	files, dirs, ignored, dots int
	totalSize                  int64 // bytes of regular files
	totalDir                   int64 // entries inside the directories we saw
}

// glyphs are the I18N table from the script.
const (
	gBroken     = "X"
	gEmpty      = "0"
	gSingle     = "1"
	gHardLink   = "<-"
	gSetUID     = "u"
	gSetGID     = "g"
	gSticky     = "s"
	gWorldWrite = "W"
	gUnreadable = "r"
	gNoSearch   = "x"
)

// readNames returns the visible names of dir, sorted (os.ReadDir sorts), hiding
// only "." and "..".
func readNames(dir string) ([]string, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(des))
	for _, de := range des {
		n := de.Name()
		if n == "." || n == ".." {
			continue
		}
		names = append(names, n)
	}
	return names, nil
}

// contentEntry is one non-dot entry of a directory, with its real type.
type contentEntry struct {
	name  string
	isDir bool
}

// dirContent lists the non-dot entries of a directory, as `glob -- *` does, but
// keeps the type that readdir already gives us (d_type) instead of guessing it
// from the name.
func dirContent(path string) ([]contentEntry, bool) {
	des, err := os.ReadDir(path)
	if err != nil {
		return nil, false
	}
	out := make([]contentEntry, 0, len(des))
	for _, de := range des {
		if strings.HasPrefix(de.Name(), ".") {
			continue
		}
		out = append(out, contentEntry{name: de.Name(), isDir: entryIsDir(path, de)})
	}
	return out, true
}

// entryIsDir answers "is this a directory" from the type readdir reported.
//
// On filesystems that report DT_UNKNOWN (some FUSE/NFS mounts) Go's DirEntry
// type is 0, which is also what a regular file gets, so the answer is only
// ambiguous when there is no suffix to go by -- and that is exactly the case
// where the old heuristic guessed. We lstat just those few names.
func entryIsDir(dir string, de fs.DirEntry) bool {
	t := de.Type()
	switch {
	case t&fs.ModeDir != 0:
		return true
	case t&fs.ModeSymlink != 0:
		// a link to a directory is a directory for classification purposes,
		// but only worth a stat when the name gives no hint
		if strings.HasSuffix(de.Name(), ".") || hasSuffix(de.Name()) {
			return false
		}
		fi, err := os.Stat(filepath.Join(dir, de.Name()))
		return err == nil && fi.IsDir()
	case t != 0:
		return false // symlink/socket/fifo handled above, other specials are not dirs
	}
	// t == 0: regular file, or a filesystem that does not tell us
	if hasSuffix(de.Name()) {
		return false
	}
	fi, err := os.Lstat(filepath.Join(dir, de.Name()))
	return err == nil && fi.Mode()&fs.ModeDir != 0
}

// hasSuffix reports whether the name ends in something that looks like a
// file extension (".tar", not a dot hidden file and not a trailing dot).
func hasSuffix(name string) bool {
	i := strings.LastIndexByte(name, '.')
	return i > 0 && i < len(name)-1
}

// buildEntry produces the data record for one name. It returns nil for entries
// that are filtered out by the ignore level.
func (c *Ctx) buildEntry(dir, name, path string, listed map[string]bool, cnt *counters, igl *[]string) *Entry {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil
	}

	etype := "file"
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		etype = "link"
	case fi.IsDir():
		etype = "directory"
	case !fi.Mode().IsRegular():
		etype = "special"
	}

	g, iglevel, ts, title, sfx := classify(dir, name, etype)

	warning := ""
	if base, ok := autoBkupBase(name); ok && !listed[base] {
		// abandoned editor/cvs backup: "foo.~1.2.~" with no "foo" left
		warning += gBroken
	} else if c.opts.Ignore >= iglevel {
		if iglevel >= 3 {
			*igl = append(*igl, name)
		}
		if iglevel == 2 {
			cnt.dots++
		} else {
			cnt.ignored++
		}
		return nil
	}

	st := statOf(fi)
	mode := st.mode

	style := ""
	displayInline := ""
	pre := "  "
	post := ""
	typec := ""
	size := int64(0)
	mtime := st.mtime
	atime := st.atime

	// a. general
	if mode&04000 != 0 {
		warning += gSetUID
	}
	if mode&02000 != 0 {
		warning += gSetGID
	}
	if mode&01000 != 0 {
		warning += gSticky
	}
	post += c.peculiar(mode, etype)
	if etype == "link" {
		typec = "->"
		style = styleMap["link"]
	}

	// b. by type
	//
	// One stat, not two. Lstat above already told us the link type, so a second
	// Lstat inside Stat() would only repeat it; Stat() is called solely for
	// symlinks, where following the link is the whole point (it is how a
	// dangling link is recognised, and Tcl's "file exists" behaved the same).
	// For everything else tgt is the Lstat result. The only behavioural
	// difference is a file that vanishes between the two calls: it used to be
	// reported as broken, and now keeps the attributes Lstat saw. That was the
	// race the second stat measured, not a property of the file.
	tgt, terr := fi, error(nil)
	if etype == "link" {
		tgt, terr = os.Stat(path)
	}
	switch {
	case etype == "link" && c.external(dir, path):
		// don't stat links that point to other volumes: they may be slow
		typec = "-->"
		if sfx == "" {
			style, g = styleMap["dir"], "dir"
			cnt.dirs++
		} else {
			cnt.files++
		}
	case terr != nil:
		warning += gBroken
		if s, ok := styleMap["broken"]; ok {
			style = s
		}
	case tgt.Mode().IsRegular():
		style = styleMap["file"]
		typec = "" // a link to a plain file shows no marker, only a broken link keeps "->"
		if isExecutable(path) {
			if g == "other" {
				g = "app"
			}
			warning = "*" + warning
			style = styleMap["exec"] // LS_COLORS ex=... when set
		}
		if etype != "link" {
			if !isReadable(path) {
				warning += gUnreadable
			}
			size = st.size
			if size == 0 {
				size = resourceForkLen(path) // macOS resource-fork fallback
			}
			cnt.totalSize += size
			if size == 0 {
				warning += gEmpty
			}
			if st.nlink > 1 {
				warning += gHardLink
			}
			// A .nib is either a NeXT/Interface Builder bundle or an Apple II
			// nibble image; the size tells them apart. 232960 = 5.25" 16-sector
			// (35 x 6656), 116480 = 5.25" 13-sector (DOS 3.2).
			// A .nib is either a NeXT/Interface Builder bundle or an Apple II
			// nibble image; the size tells them apart. 232960 = 5.25" 16-sector
			// (35 x 6656), 116480 = 5.25" 13-sector (DOS 3.2 / DOS 3.3 pre-16).
			if sfx == ".nib" && (size == 232960 || size == 116480) {
				g = "A2"
			}
			if sfx == ".doc" && size < 22000 {
				g = "doc"
			}
		}
		cnt.files++
	case tgt.IsDir():
		content, readable := dirContent(path)
		var ndir int64
		var matched int
		g, ndir, matched = classifyDir(path, name, content, c.opts.LegacyClassify)
		size = ndir
		style = styleMap["dir"]
		if etype == "link" {
			size = 0
		} else {
			typec = "/"
			cnt.totalDir += size
			switch {
			case !readable:
				warning += gUnreadable
			case size == 0:
				warning += gEmpty
			case size == 1 && exists(filepath.Join(path, "Contents", "Info.plist")):
				// an app bundle, don't nag about a single child
			case size == 1:
				warning += gSingle
				// The original had a disabled branch here ("elseif {0 && $size==1}")
				// that would show the lone child instead of a count; it is
				// implemented as --inline-singleton.
				if c.opts.InlineOne {
					if inl := inlineSingleton(path); inl != "" {
						displayInline = inl
						warning = strings.ReplaceAll(warning, gSingle, "")
						typec = ""
					}
				}
			default:
				post += itoa(size)
				// The original's TAP experiment: show how many entries actually
				// matched the guessed group ("7+15" = 7 of 22 are images).
				if c.opts.GroupMix && g != "dir" && matched > 0 && int64(matched) < size {
					post = " " + itoa(int64(matched)) + "+" + itoa(size-int64(matched))
				}
			}
			if mode&2 != 0 {
				warning += gWorldWrite
			}
			if !isExecutable(path) {
				warning += gNoSearch
			}
		}
		cnt.dirs++
	default:
		g = "special"
		style = styleMap["special"]
		typec = "_"
		cnt.files++
	}

	// the original's TAP experiment: in your home directory, name the owner of
	// files that are not yours (root's leftovers, another user's mount, ...)
	if c.opts.OwnerNotes && etype == "file" && c.isHome(dir) && st.uid != uint32(os.Getuid()) && st.nlink == 1 {
		if who := userName(st.uid); who != "" {
			post += " owner=" + who
		}
	}

	// c. appearance
	display := name
	if displayInline != "" {
		display = name + "/" + displayInline
	}
	sortkey := normalizeKey(title, sfx)
	if c.opts.Title {
		display = title
		if groupBy[g] != nil && groupBy[g].title {
			display = prettyName(display)
		}
		if c.opts.NoSfx && groupBy[g] != nil && groupBy[g].nosfx && etype == "file" {
			sortkey = tclRoot(sortkey)
		} else {
			display += sfx
		}
	}
	if c.opts.F {
		display += typec
	}
	if c.opts.Notable && notableName(ts) {
		style += styleMap["notable"]
	}
	// LS_COLORS per-extension colours win over everything except warnings
	if s, ok := styleForExt(name); ok {
		style = s
	}
	if c.batchAnn != nil {
		post += c.batchAnn[path]
	} else {
		post += c.annotate(dir, path, name)
	}

	return &Entry{
		Tail: name, Sfx: sfx, Type: etype, Group: g, Sortkey: sortkey,
		Style: style, Pre: pre, Display: display, Warning: warning, Post: post,
		Size: size, Mtime: mtime, Atime: atime,
	}
}

// isHome reports whether dir is the user's home directory (the original only
// annotates owners there: it is meant to spot leftovers from root or guests).
func (c *Ctx) isHome(dir string) bool {
	if c.home == "" {
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	return abs == c.home
}

// userName maps a numeric uid to a login name, "" if unknown.
func userName(uid uint32) string {
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return ""
	}
	return u.Username
}

// inlineSingleton returns "child" or "child/grandchild" when the directory
// holds exactly one visible entry, so that "src/1" can be shown as
// "src/main.c". Depth is capped: after three levels the name gets too long to
// be useful, which is why the original left this switched off.
func inlineSingleton(path string) string {
	out := ""
	cur := path
	for depth := 0; depth < 3; depth++ {
		names, err := readNames(cur)
		if err != nil {
			break
		}
		var visible []string
		for _, n := range names {
			if !strings.HasPrefix(n, ".") {
				visible = append(visible, n)
			}
		}
		if len(visible) != 1 {
			break
		}
		out = filepath.Join(out, visible[0])
		fi, err := os.Lstat(filepath.Join(cur, visible[0]))
		if err != nil || !fi.IsDir() {
			break
		}
		cur = filepath.Join(cur, visible[0])
	}
	return out
}

// external reports whether the link leaves the current directory for a mount
// point that may need to spin up (the original watched /Volumes).
func (c *Ctx) external(dir, path string) bool {
	link, err := os.Readlink(path)
	if err != nil {
		return false
	}
	if strings.HasPrefix(link, dir) {
		return false
	}
	for _, m := range c.opts.Mounts {
		if m == "" || !strings.HasPrefix(link, m+"/") {
			continue
		}
		// the volume has to be there: an unmounted /media/usb link is just broken
		rest := strings.SplitN(strings.TrimPrefix(link, m+"/"), "/", 2)
		if fi, err := os.Stat(m + "/" + rest[0]); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// classifyDir guesses what a directory is by looking at its contents: a folder
// full of .mp3 is "audio", a folder with index.html is a web root, and so on.
func classifyDir(dir, show string, content []contentEntry, legacy bool) (group string, size int64, matched int) {
	const maxExt = len(".torrent") // longer "suffixes" are not suffixes

	cc := map[string]int{}
	llen := 0
	softignore := 0
	for _, ce := range content {
		name := ce.name
		g, iglevel, _, _, sfx := classify(dir, name, "file")
		if iglevel <= 2 {
			continue
		}
		if iglevel == 3 {
			softignore++
		}
		if g == "other" {
			switch {
			case ce.isDir:
				// the real thing: readdir told us it is a directory
				g = "dir"
			case legacy && (len(sfx) == 0 || len(sfx) > maxExt):
				// --legacy-classify: the Tcl rule, "no suffix, so probably
				// a directory". Wrong for README, Makefile, LICENSE, ...
				g = "dir"
			}
		}
		cc[g]++
		llen++
	}

	known := llen - cc["other"]
	max, maxg := 0, "dir"
	for _, g := range groupOrder {
		if g == "other" || g == "A2" || g == "ignore" {
			continue
		}
		if cc[g] > max {
			max, maxg = cc[g], g
		}
	}
	relen := cc["video"] + cc["audio"] + cc["image"] + cc["WWW"]

	g := "dir"
	switch {
	case strings.HasPrefix(show, "."):
		g = "dotfile"
	case containsVideoTS(content):
		g = "video"
	case cc["dir"] >= 3:
		// a hierarchy, keep it as a directory
	case relen > 0 && (relen >= llen/2 || llen <= 4):
		// media folder: pick the dominant kind
		switch {
		case cc["video"] > 0:
			g = "video"
		case cc["audio"] > cc["image"]:
			g = "audio"
		case cc["WWW"] >= 2 || (cc["WWW"] == 1 && relen <= 2) || (cc["WWW"] >= 1 && hasIndexHTML(content)):
			// keep it a directory: an .html folder looks like a web root
		default:
			g = "image"
		}
	case legacy && known > 0 && max >= known/2:
		// the Tcl rule: half of the recognised entries decide
		if maxg != "WWW" {
			g = maxg
		}
	case !legacy && max >= 2 && max*2 >= known:
		// Half of the recognised entries, but only from two matching files
		// up. The Tcl rule let a SINGLE file decide: a package doc directory
		// holding just "README.gz" and "copyright" became "build", because
		// "copyright" was not counted as a directory any more.
		if maxg != "WWW" {
			g = maxg
		}
	}

	if devDirName(show) {
		g = "dev"
	}

	size = int64(llen - softignore)
	if size < 0 {
		size = 0
	}
	return g, size, cc[g]
}

func containsVideoTS(content []contentEntry) bool {
	for _, ce := range content {
		if videoTSName(ce.name) {
			return true
		}
	}
	return false
}

func hasIndexHTML(content []contentEntry) bool {
	for _, ce := range content {
		if strings.HasPrefix(strings.ToLower(ce.name), "index.htm") {
			return true
		}
	}
	return false
}
