package main

// Command line parsing.
//
// Two spellings are accepted for every option:
//
//	--long-option        GNU style, recommended
//	-legacyLong          the Tcl script's spelling, REMOVED in favour of --long
//	                     of muscle memory ("sl -only", "sl -width 100") keeps working
//	-x                   short form for the handful of options typed daily
//
// Values may be given as --opt=value, --opt value, -o value or -ovalue.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// version continues the original's version line: the Tcl script went up to
// v1.1.3 (revision 1.89, February 2012), so the Go port is the next patch
// release rather than a new 2.x. Do not bump it without being asked.
const version = "1.1.4"

type optSpec struct {
	long   string
	short  byte
	hasArg bool   // the value is required and may be a separate word
	optArg bool   // the value is optional and must be attached with "="
	arg    string // placeholder for the help text
	help   string
	apply  func(o *Options, val string) error
}

// setColor implements --color[=WHEN] and the colour key of the config file.
// The internal representation stays an int: 1 = always, 0 = never, -1 = auto,
// because that is what the rest of the program (and NO_COLOR) already speaks.
func setColor(o *Options, v string) error {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "always", "on", "yes", "true", "1", "force", "colour", "color":
		o.Color = 1
	case "auto", "tty", "if-tty", "default", "-1":
		o.Color = -1
	case "never", "off", "no", "false", "0", "none":
		o.Color = 0
	default:
		return fmt.Errorf("expected auto, always or never, got %q", v)
	}
	return nil
}

func intOpt(dst func(o *Options) *int) func(*Options, string) error {
	return func(o *Options, v string) error {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("expected a number, got %q", v)
		}
		*dst(o) = n
		return nil
	}
}

func int64Opt(dst func(o *Options) *int64) func(*Options, string) error {
	return func(o *Options, v string) error {
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return fmt.Errorf("expected a number, got %q", v)
		}
		*dst(o) = n
		return nil
	}
}

func strOpt(dst func(o *Options) *string) func(*Options, string) error {
	return func(o *Options, v string) error { *dst(o) = v; return nil }
}

func boolOpt(f func(o *Options, v bool)) func(*Options, string) error {
	return func(o *Options, v string) error {
		b := true
		switch strings.ToLower(v) {
		case "0", "off", "no", "false", "never":
			b = false
		case "", "1", "on", "yes", "true", "always", "auto":
			b = true
		default:
			return fmt.Errorf("expected on/off, got %q", v)
		}
		f(o, b)
		return nil
	}
}

var optionSpecs []optSpec

func init() {
	optionSpecs = []optSpec{
		{long: "all", short: 'a', help: "show dotfiles and normally ignored files (ignore level 0)",
			apply: func(o *Options, _ string) error { o.Ignore = 0; return nil }},
		{long: "one", short: '1', help: "one entry per line (--one=off turns it back)",
			apply: boolOpt(func(o *Options, b bool) { o.One = b })},
		{long: "only", short: 'o', help: "show only entries that carry some signal",
			apply: func(o *Options, _ string) error {
				o.Only = true
				o.RelSize, o.Series, o.Prefix, o.Title, o.Summary = 0, 0, 0, false, 0
				return nil
			}},
		{long: "title", short: 't', help: "prettify names (camelCase -> words, separators -> space)",
			apply: boolOpt(func(o *Options, b bool) { o.Title = b })},
		{long: "nosfx", help: "with --title, drop the suffix for audio/video/image/font",
			apply: boolOpt(func(o *Options, b bool) { o.NoSfx = b })},
		{long: "width", short: 'w', hasArg: true, arg: "N", help: "assume a terminal N columns wide",
			apply: intOpt(func(o *Options) *int { return &o.Width })},
		{long: "prefix", hasArg: true, arg: "N", help: "elide a shared prefix of at least N chars (0 = off)",
			apply: intOpt(func(o *Options) *int { return &o.Prefix })},
		{long: "series", hasArg: true, arg: "N", help: "collapse runs of at least N numbered names (default 5)",
			apply: intOpt(func(o *Options) *int { return &o.Series })},
		{long: "ignore", hasArg: true, arg: "N", help: "ignore level to show: 3 default, 2 +.h/.o, 1 +dotfiles, 0 all",
			apply: intOpt(func(o *Options) *int { return &o.Ignore })},
		{long: "relsize", hasArg: true, arg: "N", help: "percent of entries called \"relatively large\" (0 = off)",
			apply: intOpt(func(o *Options) *int { return &o.RelSize })},
		{long: "relchange", hasArg: true, arg: "N", help: "percent of entries called \"relatively recent\" (0 = off)",
			apply: intOpt(func(o *Options) *int { return &o.RelChange })},
		{long: "relread", hasArg: true, arg: "N", help: "percent of entries called \"relatively read\" (0 = off)",
			apply: intOpt(func(o *Options) *int { return &o.RelRead })},
		{long: "atime", hasArg: true, arg: "N", help: "also mark files read in the last N seconds (alias --age)",
			apply: int64Opt(func(o *Options) *int64 { return &o.Atime })},
		{long: "age", hasArg: true, arg: "N", help: "alias for --atime (the name used on the original site)",
			apply: int64Opt(func(o *Options) *int64 { return &o.Atime })},
		{long: "summary", hasArg: true, arg: "N", help: "print the summary line from N entries on (0 = never)",
			apply: intOpt(func(o *Options) *int { return &o.Summary })},
		{long: "expando", hasArg: true, arg: "N", help: "auto-expand a directory with at most N files (0 = off)",
			apply: intOpt(func(o *Options) *int { return &o.Expando })},
		{long: "notable", help: "highlight README/TODO/Makefile/... (default on)",
			apply: boolOpt(func(o *Options, b bool) { o.Notable = b })},
		{long: "no-notable", help: "do not highlight README/TODO/Makefile/...",
			apply: func(o *Options, _ string) error { o.Notable = false; return nil }},
		{long: "k1000", help: "use 1000, not 1024, as the size divisor (--k1000=off puts it back)",
			apply: boolOpt(func(o *Options, b bool) {
				if b {
					o.Base = 1000
				} else {
					o.Base = 1024
				}
			})},
		{long: "k1024", help: "use 1024 as the size divisor (default)",
			apply: func(o *Options, _ string) error { o.Base = 1024; return nil }},
		{long: "colsep", hasArg: true, arg: "N", help: "minimal gap between columns (default 1)",
			apply: intOpt(func(o *Options) *int { return &o.ColSep })},
		{long: "namemin", hasArg: true, arg: "N", help: "width that still allows two columns (default 40)",
			apply: intOpt(func(o *Options) *int { return &o.NameMin })},
		{long: "shorten", hasArg: true, arg: "N", help: "start shortening names when there are at least N entries",
			apply: intOpt(func(o *Options) *int { return &o.Shorten })},
		{long: "ellipsis", hasArg: true, arg: "S", help: "text used when a name is shortened (default \"...\")",
			apply: strOpt(func(o *Options) *string { return &o.Ellipsis })},
		{long: "owner-notes", help: "in the home directory, name the owner of files that are not yours (default on)",
			apply: boolOpt(func(o *Options, b bool) { o.OwnerNotes = b })},
		{long: "no-owner-notes", help: "do not annotate files owned by other users",
			apply: func(o *Options, _ string) error { o.OwnerNotes = false; return nil }},
		{long: "group-mix", help: "print \"5+3\" instead of \"8\" on a directory classified by its contents",
			apply: boolOpt(func(o *Options, b bool) { o.GroupMix = b })},
		{long: "experimental", help: "alias for --group-mix plus --owner-notes (the original's TAP branch)",
			apply: func(o *Options, _ string) error { o.GroupMix, o.OwnerNotes = true, true; return nil }},
		{long: "inline-singleton", help: "show \"src/main.c\" instead of \"src/1\" for a directory holding one entry",
			apply: boolOpt(func(o *Options, b bool) { o.InlineOne = b })},
		{long: "legacy-classify", help: "Tcl rule: an entry with no suffix inside a directory counts as a subdirectory",
			apply: boolOpt(func(o *Options, b bool) { o.LegacyClassify = b })},
		{long: "legacy-series", help: "Tcl rule: names that differ in any one digit or letter count as a series",
			apply: boolOpt(func(o *Options, b bool) { o.LegacySeries = b })},
		{long: "log", short: 'l', help: "print layout diagnostics",
			apply: func(o *Options, _ string) error { o.Log = true; return nil }},
		{long: "group", help: "group by kind (default on)",
			apply: boolOpt(func(o *Options, b bool) { o.Group = b })},
		{long: "nogroup", help: "one flat sorted list, no group titles",
			apply: func(o *Options, _ string) error { o.Group, o.GroupTitle = false, false; return nil }},
		{long: "color", short: 'c', optArg: true, arg: "WHEN",
			help:  "colour: auto (default), always, never; a bare -c/--color means always",
			apply: func(o *Options, v string) error { return setColor(o, v) }},
		{long: "no-color", help: "never use colors",
			apply: func(o *Options, _ string) error { o.Color = 0; return nil }},
		{long: "annotate", hasArg: true, arg: "CMD", help: "run a shell command per entry and show its output (see --help)",
			apply: strOpt(func(o *Options) *string { return &o.Annotate })},
		{long: "annotate-batch", hasArg: true, arg: "CMD",
			help:  "run CMD once per directory; it gets the paths on stdin, one output line per path",
			apply: strOpt(func(o *Options) *string { return &o.AnnotateBatch })},
		{long: "annotate-match", hasArg: true, arg: "RE", help: "only annotate entries whose name matches RE",
			apply: strOpt(func(o *Options) *string { return &o.AnnotateMatch })},
		{long: "mounts", hasArg: true, arg: "LIST", help: "mount points that may spin up (colon separated)",
			apply: func(o *Options, v string) error {
				o.Mounts = strings.FieldsFunc(v, func(r rune) bool { return r == ':' || r == ',' })
				return nil
			}},
		{long: "rule", hasArg: true, arg: "RE -> GROUP",
			help:  "force a group when the name matches RE; repeatable, first match wins",
			apply: func(o *Options, v string) error { o.Rules = append(o.Rules, v); return nil }},
		{long: "autosearch", hasArg: true, arg: "RE", help: "highlight names matching RE (default: urgent|password|pw|live)",
			apply: strOpt(func(o *Options) *string { return &o.Autosearch })},
		{long: "config", hasArg: true, arg: "FILE", help: "read configuration from FILE, which must exist (default: $XDG_CONFIG_HOME/sl/config, created on first run)",
			apply: strOpt(func(o *Options) *string { return &o.ConfigFile })},
		{long: "no-config", help: "do not read (or create) any configuration file",
			apply: func(o *Options, _ string) error { o.NoConfig = true; return nil }},
		{long: "version", short: 'V', help: "print the version and exit",
			apply: func(o *Options, _ string) error { printVersion(); os.Exit(0); return nil }},
		{long: "help", short: 'h', help: "print this help and exit",
			apply: func(o *Options, _ string) error { usage(os.Stdout); os.Exit(0); return nil }},
		{long: "startup", hasArg: true, arg: "FILE", help: "(ignored) the Tcl startup file; use --config instead",
			apply: func(o *Options, v string) error {
				fmt.Fprintf(os.Stderr, "sl: --startup %s ignored: the Go port does not run Tcl; use --config\n", v)
				return nil
			}},
	}
}

var (
	longSpecs  = map[string]*optSpec{}
	shortSpecs = map[byte]*optSpec{}
)

func init() {
	for i := range optionSpecs {
		s := &optionSpecs[i]
		longSpecs[s.long] = s
		if s.short != 0 {
			shortSpecs[s.short] = s
		}
	}
}

func printVersion() {
	fmt.Printf("sl %s\n", version)
	fmt.Printf("a Go port of the Tcl script sl 1.1.3 by Thomas A. Phelps\n")
	fmt.Printf("  original: http://practicalthought.com/sl/ (offline since ~2017,\n")
	fmt.Printf("            archived at https://web.archive.org/web/2016/http://practicalthought.com/sl/)\n")
	fmt.Printf("  port:     goberghen & Arena.ai agents, GPL-3.0-or-later\n")
}

// parseArgs understands "--long", "-x", with values as "=v", " v", "-xv" or
// "--long=v". There is no "-long" form: one dash means a short option, and
// nothing else (the Tcl version's spelling was dropped -- it turned a typo into
// a silent misparse).
// unknownOpt turns "unknown option: -x" into something actionable: if the name
// exists as a long option, say so. Guessing that "-only" means "-o -n -l -y"
// would be worse than refusing.
func unknownOpt(a string) error {
	if len(a) > 1 && !strings.HasPrefix(a, "--") {
		name := strings.TrimPrefix(a, "-")
		if k := strings.IndexByte(name, '='); k >= 0 {
			name = name[:k]
		}
		if longSpecs[name] != nil {
			return fmt.Errorf("unknown option: %s (did you mean --%s?)", a, name)
		}
	}
	return fmt.Errorf("unknown option: %s", a)
}

func parseArgs(argv []string) (*Options, []string, error) {
	o := defaultOptions()
	var files []string

	for i := 0; i < len(argv); i++ {
		a := argv[i]
		switch {
		case a == "--":
			return o, append(files, argv[i+1:]...), nil
		case a == "-" || !strings.HasPrefix(a, "-"):
			return o, append(files, argv[i:]...), nil
		}

		var spec *optSpec
		var val string
		hasVal := false

		if strings.HasPrefix(a, "--") {
			name := a[2:]
			if k := strings.IndexByte(name, '='); k >= 0 {
				name, val, hasVal = name[:k], name[k+1:], true
			}
			spec = longSpecs[name]
			if spec == nil {
				return nil, nil, fmt.Errorf("unknown option: %s", a)
			}
		} else if len(a) == 2 {
			// a single letter: a short option, nothing else. The Tcl version
			// also allowed "-width", i.e. a long name behind one dash; that is
			// gone (see the note at the top of the file), and the error says
			// what to write instead rather than guessing what was meant.
			spec = shortSpecs[a[1]]
			if spec == nil {
				return nil, nil, unknownOpt(a)
			}
		} else {
			// -w80 : a short option with its value attached. This is ordinary
			// getopt behaviour and stays.
			name := a[1:]
			if k := strings.IndexByte(name, '='); k >= 0 {
				name, val, hasVal = name[:k], name[k+1:], true
			}
			if longSpecs[name] != nil {
				// "-width": a long name behind one dash. Never guess that this
				// means "-w -i -d ..."; say what to write instead.
				return nil, nil, unknownOpt(a)
			}
			if s := shortSpecs[a[1]]; s != nil && s.hasArg {
				spec, val, hasVal = s, a[2:], true
			} else {
				return nil, nil, unknownOpt(a)
			}
		}

		if spec.hasArg && !hasVal {
			if i+1 >= len(argv) {
				return nil, nil, fmt.Errorf("option %s needs a value", a)
			}
			i++
			val = argv[i]
		} else if spec.optArg && !hasVal {
			// an optional value must be attached ("--color=never"); a separate
			// word would be ambiguous with a file name, so it is not consumed
			val = ""
		}
		if err := spec.apply(o, val); err != nil {
			if spec.long == "annotate-match" || spec.long == "autosearch" {
				return nil, nil, fmt.Errorf("%s: %v", spec.long, err)
			}
			return nil, nil, fmt.Errorf("%s: %v", spec.long, err)
		}
	}
	return o, files, nil
}

func usage(w *os.File) {
	fmt.Fprintf(w, `NAME
       sl - usefully organize directories and point out interesting files

SYNOPSIS
       sl [OPTION]... [FILE|DIRECTORY]...

       Files are grouped by kind (dir, dev, WWW, doc, image, audio, video,
       archive, data, A2, ...), sorted inside each group, laid out in columns
       and marked when they stand out from the rest of the directory: large,
       recently changed, recently read, empty, unreadable, world-writable,
       hard-linked, broken link, odd permissions, or part of a name series.

OPTIONS
       Options are spelled the GNU way: two dashes for a long name (--width),
       one dash for a short one (-w). Values may be written as --width=100,
       --width 100, -w 100 or -w100.

       The original Tcl script also accepted the long names behind a single
       dash (-width 100). This port does not: one dash always means a short
       option, and nothing else. A Tcl-style spelling is rejected with a hint
       rather than guessed at:

              sl: unknown option: -only (did you mean --only?)

`)
	for _, s := range optionSpecs {
		if s.long == "startup" {
			continue
		}
		head := "       "
		if s.short != 0 {
			head += fmt.Sprintf("-%c, ", s.short)
		} else {
			head += "    "
		}
		head += "--" + s.long
		switch {
		case s.hasArg:
			head += " " + s.arg
		case s.optArg:
			head += "[=" + s.arg + "]"
		}
		fmt.Fprintf(w, "%-34s %s\n", head, s.help)
	}
	fmt.Fprintf(w, `
ANNOTATION
       %s

       /     directory                          *     executable
       ->    symbolic link (--> off-volume)     _     special file
       0     empty file or empty directory      1     directory with one entry
       <-    more than one hard link            X     broken link, stale backup
       u g s setuid/setgid/sticky               W     world-writable
       r     not readable                       x     directory not searchable
       <2h   changed recently (top %% of dir)    -5m   read recently
       4.2M  relatively large                   ...7  run of 7 names collapsed

COMPATIBILITY
       Two rules of the Tcl original are fixed by default and can be restored:
         --legacy-classify   no suffix inside a directory counts as a subdirectory
         --legacy-series     names differing in any one digit or letter are a series
       --owner-notes (on) and --group-mix (off) are the original's TAP branch.

CONFIGURATION
       Defaults are read from $XDG_CONFIG_HOME/sl/config (usually
       ~/.config/sl/config). On the first run sl creates that file with every
       setting commented out and the default value shown, so it changes
       nothing until you edit it; an existing file is never overwritten.
       Command line options win over the file; --no-config skips both.

FILES
       $XDG_CONFIG_HOME/sl/config      user configuration

ORIGINAL
       sl stands on the shoulders of Thomas A. Phelps' Tcl script "sl -
       usefully organize directories and point out interesting files"
       (v1.1.3, revision 1.89, February 2012, GPLv3). Its home page,
       http://practicalthought.com/sl/, is offline since about 2017 and is
       preserved at https://web.archive.org/web/2016/http://practicalthought.com/sl/

       This Go port keeps that behaviour; see INSTALL.md for the differences.
       In Debian it is packaged as sl-go and installed as /usr/bin/sl; it
       declares Conflicts: sl, so it cannot share a system with the locomotive.
`, "Warning glyphs:")
}
