package main

// First-run configuration: if the user has no config file yet, write one that
// documents every setting with its built-in default.
//
// The template is a Go constant rather than an embedded file so that the
// binary stays a single artifact with no build-time inputs. Every setting in
// it is commented out: an unmodified file must parse to exactly the built-in
// defaults, so that "there is a config file" and "there is no config file"
// differ only in the comments the user can read.

import (
	"fmt"
	"os"
	"path/filepath"
)

const configTemplate = `# sl configuration
#
# Read on every run from $XDG_CONFIG_HOME/sl/config (normally
# ~/.config/sl/config), and created with these comments on first run.
# Format: one "key = value" per line, '#' starts a comment.
# Every long option of the program works as a key; command line options win
# over this file.
#
# Everything below is commented out, so this file changes nothing until you
# uncomment a line. Run "sl --help" for the full option list and "man sl" for
# the details.

# ---------------------------------------------------------------- what to show
# ignore = 3          # 3 = default, 2 = also .h/.o, 1 = also dotfiles, 0 = all
# title = off         # prettify names: camelCase -> words, separators -> space
# nosfx = off         # with title: drop suffixes for audio/video/image/font
# notable = on        # highlight README, TODO, Makefile, ...
# autosearch = (?i)\W(urgent|password|pw|live)\W   # names to highlight in red

# ------------------------------------------------------------- what to condense
# series = 5          # collapse runs of at least N numbered names (0 = off)
# sort = name         # order inside a group: name, size, time, atime
# prefix = 0          # elide a common prefix of at least N chars (0 = off)
# summary = 2         # print the summary line from N entries on (0 = never)
# expando = 2         # also list the single subdirectory if there are <= N files

# --------------------------------------------------------------- what to mark
# relsize = 5         # percent of entries called "relatively large"
# relchange = 10      # percent of entries called "relatively recent"
# relread = 10        # percent of entries called "relatively read"
# atime = 0           # also mark files read in the last N seconds (0 = off)
# owner-notes = on    # in the home directory, name the owner of foreign files
# group-mix = off     # show "5+3" instead of "8" on a directory classified by content
# legacy-classify = off   # Tcl rule: an entry with no suffix counts as a subdirectory
# legacy-series = off     # Tcl rule: names differing in any one digit or letter are a series
# inline-singleton = off  # show "src/main.c" instead of "src/1"

# --------------------------------------------------------------------- layout
# one = off           # one entry per line; off = as many columns as fit
# width = 0           # terminal width; 0 = ask the terminal, then $COLUMNS, then 80
# colsep = 1          # minimal gap between columns
# namemin = 40        # width that still leaves room for two columns
# shorten = 10        # start shortening names when there are at least N entries
# ellipsis = ...      # text used when a name is shortened
# k1000 = off         # use 1000, not 1024, as the size divisor (k1024 = on is the default)
# color = auto        # auto (default), always, never
# group = on          # group by kind (nogroup = on gives one flat list)
# mounts = /Volumes:/run/media:/media:/mnt   # volumes that may need to spin up

# ----------------------------------------------------------------- user rules
# Force a group when a name matches a regular expression. This is what people
# used the Tcl original's ~/.sl.tcl for, without an interpreter.
#
#   rule = <regexp> -> <group>
#
# The regexp is Go syntax; it is unanchored and case-sensitive unless it starts
# with (?i). It is matched against the classification name: the entry's name
# with a compression wrapper stripped, so "foo.1.gz" is seen as "foo.1".
# Rules are tried in order and the first match wins. A rule overrides the
# built-in tables (suffixes, exact names, README/Makefile detection). Naming the
# group "ignore" hides the entry, the same way built-in ignorables are hidden
# (visible again with --all).
#
# The two lines below are examples, not defaults:
# rule = \.tmp$ -> ignore
# rule = ^IMG_[0-9]+\.JPG$ -> image
# Groups you may name: special dir build app dev A2 WWW aux doc book document
# image vector audio video font data other archive dotfile Mac file ignore

# -------------------------------------------------------------------- colours
# SGR attributes, without the leading ESC[ ; "bold,blue" also works
# style.dir = 01;34
# style.file = 0
# style.link = 36
# style.exec = 01;32
# style.broken = 01;31
# style.special = 35
# style.group = 01;35;4
# style.warning = 01;31
# style.notable = 01;31

# ----------------------------------------------------------- extra extensions
# Add suffixes (or exact names) to an existing group; a leading "-" means
# "belongs there, but stay hidden unless --all".
# The two lines below are examples, not defaults:
# group.doc.extra = .md .rst
# group.dev.extra = .go .rs .zig

# ----------------------------------------------------------------------- hooks
# Run a command per entry and append the first line of its output.
# $1 is the path, $2 is the name.
# The four lines below are examples, not defaults:
# annotate = git log -1 --format=%cr -- "$1"
# annotate-match = \.go$
# One command per directory instead of one per entry: it reads the paths from
# stdin, one per line, and prints one line per path.
# annotate-batch = while read -r p; do git log -1 --format=%cr -- "$p"; done
`

// ensureUserConfig writes the template to path if nothing is there yet.
//
// It returns the path it created, or "" if there was nothing to do. Errors are
// returned separately: a read-only home directory is not a reason to refuse to
// list files, so callers may ignore them.
func ensureUserConfig(path string, quiet bool) (string, error) {
	if path == "" {
		return "", nil
	}
	if _, err := os.Lstat(path); err == nil {
		return "", nil // already there: never overwrite a user's file
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		// lost a race with another sl, or no permission: that is fine
		if os.IsExist(err) {
			return "", nil
		}
		return "", err
	}
	if _, err := f.WriteString(configTemplate); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if !quiet {
		fmt.Fprintf(os.Stderr, "sl: created %s (edit it, or run with --no-config)\n", path)
	}
	return path, nil
}
