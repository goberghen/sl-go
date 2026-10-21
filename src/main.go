package main

// sl - usefully organize directories and point out interesting files
//
// A Go port of Tom Phelps' Tcl script sl 1.1.3 (revision 1.89, February 2012),
// originally at http://practicalthought.com/sl/ (offline since about 2017,
// archived at https://web.archive.org/web/2016/http://practicalthought.com/sl/).
//
// The data pipeline is the same as in the original:
//
//	collect -> mark up -> order -> lay out -> print

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

type Options struct {
	Ignore         int   // entries whose ignore level is <= this are hidden (--all sets 0)
	Only           bool  // show only entries that carry some signal
	Atime          int64 // absolute "read recently" window in seconds
	One            bool  // one entry per line
	Width          int   // override the terminal width
	Log            bool  // explain what the layout code did
	Color          int   // -1 auto, 0 off, 1 on
	Summary        int   // print the summary line when at least this many entries
	Group          bool  // group by kind
	GroupTitle     bool  // print group titles
	Series         int   // collapse runs of at least this many numbered names
	Prefix         int   // elide common prefixes of at least this length
	Shorten        int   // start shortening names when there are at least this many
	RelSize        int   // percent of entries to mark as "relatively large"
	RelChange      int   // percent of entries to mark as "relatively recent"
	RelRead        int   // percent of entries to mark as "relatively accessed"
	AbsChange      int64 // absolute "modified recently" window in seconds
	Expando        int   // auto-expand a directory with this few files
	Notable        bool  // highlight README/TODO/Makefile/...
	Title          bool  // show prettified names
	NoSfx          bool  // drop suffixes for media groups when --title is on
	F              bool  // append "/" "->" "_" type markers
	MaxCol         bool  // never trade a column for a row
	InlineOne      bool  // show "src/main.c" instead of "src/1" for a directory with one entry
	OwnerNotes     bool  // in $HOME, name the owner of files that are not yours
	GroupMix       bool  // show "5+3" instead of "8" on a directory classified by content
	LegacyClassify bool  // Tcl rule: a content entry with no suffix counts as a directory
	LegacySeries   bool  // Tcl rule: names differing in any one digit or letter are a series
	ColSep         int   // minimal gap between columns (COLSEP, default 1)
	NameMin        int   // width that still allows two columns (NAMEMIN, default 40)
	Base           int   // 1024 or 1000 (K, the size divisor)
	Ellipsis       string
	Mounts         []string
	Autosearch     string
	Rules          []string // --rule / "rule =" : "RE -> GROUP", in order

	// external hooks, replacing the original's Tcl callbacks
	Annotate      string // shell command run per entry; its output is appended
	AnnotateBatch string // shell command run once per directory, paths on stdin
	AnnotateMatch string // only annotate entries whose name matches this regexp

	ConfigFile string
	NoConfig   bool
}

func defaultOptions() *Options {
	return &Options{
		Ignore:     3,
		Color:      -1,
		Summary:    2,
		Group:      true,
		GroupTitle: true,
		Series:     5,
		Shorten:    10,
		RelSize:    5,
		RelChange:  10,
		RelRead:    10,
		AbsChange:  2 * secDay,
		Expando:    2,
		F:          true,
		Notable:    true,
		ColSep:     1,
		OwnerNotes: true,
		NameMin:    40,
		Base:       1024,
		Ellipsis:   "...",
		Mounts:     []string{"/Volumes", "/run/media", "/media", "/mnt"},
		Autosearch: `(?i)\W(urgent|password|pw|live)\W`,
	}
}

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

func termWidth(fd uintptr) int {
	ws := winsize{}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd,
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno == 0 && ws.Col > 0 {
		return int(ws.Col)
	}
	return 0
}

func main() {
	opts, files, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "sl: %v\n", err)
		fmt.Fprintf(os.Stderr, "sl: try --help\n")
		os.Exit(1)
	}

	// Configuration: defaults < config file < environment < command line.
	//
	// With no config file the program writes one on the first run: a commented
	// template in which every setting shows its built-in default, so the file
	// changes nothing until the user uncomments something. --no-config skips
	// both reading and creating it; --config FILE names a file that must exist.
	if !opts.NoConfig {
		path := opts.ConfigFile
		explicit := path != ""
		if path == "" {
			path = defaultConfigPath()
			if path != "" {
				// a failure to create it is not fatal: an unwritable home
				// directory must not stop "sl" from listing a directory
				// the one-line notice only makes sense for a human at a terminal
				quiet := termWidth(os.Stderr.Fd()) == 0
				if _, err := ensureUserConfig(path, quiet); err != nil && opts.Log {
					fmt.Fprintf(os.Stderr, "sl: %v\n", err)
				}
			}
		}
		if path != "" {
			if err := loadConfig(path, explicit, opts); err != nil {
				fmt.Fprintf(os.Stderr, "sl: %v\n", err)
				os.Exit(1)
			}
		}
	}
	applyEnvColors()

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	home, _ := os.UserHomeDir()
	c := &Ctx{opts: opts, w: w, errw: os.Stderr, now: time.Now(), home: home}

	// terminal width: --width, then the tty, then $COLUMNS, then 80
	if opts.Width > 0 {
		c.width = opts.Width
	} else if tty := termWidth(os.Stdout.Fd()); tty > 0 {
		c.width = tty
	} else if env := os.Getenv("COLUMNS"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			c.width = v
		}
	}
	if c.width == 0 {
		c.width = 80
	}

	switch {
	case opts.Color == 1:
		c.colorOn = true
	case opts.Color == 0:
		c.colorOn = false
	default:
		_, noColor := os.LookupEnv("NO_COLOR")
		c.colorOn = termWidth(os.Stdout.Fd()) > 0 && !noColor
	}

	if err := compileRules(opts.Rules); err != nil {
		fmt.Fprintf(os.Stderr, "sl: %v\n", err)
		os.Exit(1)
	}

	if opts.Autosearch != "" {
		re, err := regexp.Compile(opts.Autosearch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sl: autosearch: %v\n", err)
			os.Exit(1)
		}
		c.autoRe = re
		c.autoFast = opts.Autosearch == defaultAutosearch
	}

	if err := c.prepareHooks(); err != nil {
		fmt.Fprintf(os.Stderr, "sl: %v\n", err)
		os.Exit(1)
	}

	listArgs(c, files)
	w.Flush()
}
