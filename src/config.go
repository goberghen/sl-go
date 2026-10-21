package main

// Configuration file: $XDG_CONFIG_HOME/sl/config (usually ~/.config/sl/config).
//
// The Tcl original loaded ~/.sl.tcl, i.e. arbitrary Tcl code that could
// redefine any procedure. That is a security and portability problem, so here
// the configuration is plain data:
//
//	# a comment
//	ignore = 3
//	relread = 0
//	title = on
//	style.dir = 01;34
//	style.exec = 01;32
//	group.doc.extra = .md .rst
//	annotate = git log -1 --format=%%cr
//	annotate-match = \.c$
//
// Anything that was a "hook" (perfile, vc-*) is expressed as an external
// command (annotate), which is the one thing a static binary can offer instead
// of an embedded interpreter.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// defaultConfigPath returns $XDG_CONFIG_HOME/sl/config (or ~/.config/sl/config).
func defaultConfigPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "sl", "config")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "sl", "config")
}

// loadConfig applies a configuration file to opts. A missing default file is
// not an error; a missing file named explicitly on the command line is.
func loadConfig(path string, explicit bool, opts *Options) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return fmt.Errorf("%s:%d: expected key = value", path, lineNo)
		}
		key := strings.TrimSpace(line[:eq])
		val := stripInlineComment(line[eq+1:])
		if err := applyConfigKey(key, val, opts); err != nil {
			return fmt.Errorf("%s:%d: %v", path, lineNo, err)
		}
	}
	return sc.Err()
}

// applyConfigKey understands the CLI option names plus style.* and group.*.extra.
func applyConfigKey(key, val string, opts *Options) error {
	switch {
	case key == "style" || strings.HasPrefix(key, "style."):
		name := strings.TrimPrefix(strings.TrimPrefix(key, "style"), ".")
		if name == "" {
			return fmt.Errorf("style needs a name, e.g. style.dir = 01;34")
		}
		styleMap[name] = normalizeSGR(val)
		return nil

	case strings.HasPrefix(key, "group.") && strings.HasSuffix(key, ".extra"):
		g := strings.TrimSuffix(strings.TrimPrefix(key, "group."), ".extra")
		if groupBy[g] == nil {
			return fmt.Errorf("no such group: %s", g)
		}
		addExtensions(g, strings.Fields(val))
		return nil

	case key == "k":
		opts.Base = intOr(val, opts.Base)
		return nil

	case key == "no-notable":
		opts.Notable = false
		return nil
	}

	// everything else is spelled like the long option
	spec := longSpecs[key]
	if spec == nil {
		return fmt.Errorf("unknown setting: %s", key)
	}
	if spec.hasArg && val == "" {
		return fmt.Errorf("%s needs a value", key)
	}
	return spec.apply(opts, val)
}

// stripInlineComment removes a trailing " # ..." so that
//
//	ignore = 2       # show .h and .o
//
// works as expected (but a '#' inside quotes is kept).
func stripInlineComment(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, `"`) || strings.HasPrefix(v, "'") {
		q := v[:1]
		if end := strings.Index(v[1:], q); end >= 0 {
			return v[1 : 1+end]
		}
		return v
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, "\t#"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

func intOr(s string, def int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return def
	}
	return n
}

// normalizeSGR turns "01;34" (LS_COLORS/config spelling) into the internal
// ";01;34" form, and accepts "bold,blue" or a named style for convenience.
func normalizeSGR(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "-" || v == "none" {
		return ""
	}
	if s, ok := styleNames[strings.ToLower(v)]; ok {
		return s
	}
	// accept both "01;34" and ";01;34"
	return ";" + strings.TrimPrefix(v, ";")
}

var styleNames = map[string]string{
	"black": ";30", "red": ";31", "green": ";32", "yellow": ";33",
	"blue": ";34", "magenta": ";35", "cyan": ";36", "white": ";37",
	"bright-red": ";91", "bright-green": ";92", "bright-blue": ";94",
	"bold": ";1", "dim": ";2", "italic": ";3", "underline": ";4",
	"blink": ";5", "reverse": ";7", "strike": ";9",
}

// addExtensions appends suffixes (or exact names) to a group's table.
func addExtensions(group string, exts []string) {
	for _, x := range exts {
		if x == "" {
			continue
		}
		if strings.HasPrefix(x, "-") {
			eq[x[1:]] = "-" + group
		} else {
			eq[x] = group
		}
	}
}

var _ = regexp.MustCompile
