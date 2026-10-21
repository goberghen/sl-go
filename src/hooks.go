package main

// External hooks.
//
// The Tcl original let you redefine procedures from ~/.sl.tcl; perfile() got
// every field of every entry and vc-* added version-control marks. A static
// binary cannot host an interpreter, so the same job is done by running a
// command and showing its first line of output:
//
//	annotate = git log -1 --format=%cr
//	annotate-match = \.(c|h)$
//
// The command runs with the entry's path as $1 and its name as $2, in the
// directory being listed. Output is trimmed to one short line, so a stray
// "fatal: not a git repository" cannot wreck the layout.
//
// This is the natural replacement for the original's vc-* hooks:
//
//	annotate = git status --porcelain --ignored -- "$1" | head -1
//	annotate-match = .
//
// ...but it runs one process per matching entry, so use --annotate-match to
// keep it cheap.

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// annotateBatchTimeout is the deadline for the whole directory. The per-entry
// hook gets 2 s per entry, which for a directory of 500 names is 1000 s of
// worst case; batching makes the ceiling one process instead of 500.
const annotateBatchTimeout = 30 * time.Second

func (c *Ctx) prepareHooks() error {
	if c.opts.Annotate != "" && c.opts.AnnotateBatch != "" {
		return fmt.Errorf("--annotate and --annotate-batch cannot be used together")
	}
	if c.opts.Annotate == "" && c.opts.AnnotateBatch == "" {
		return nil
	}
	if c.opts.AnnotateMatch != "" {
		re, err := regexp.Compile(c.opts.AnnotateMatch)
		if err != nil {
			return fmt.Errorf("annotate-match: %v", err)
		}
		c.annotateRe = re
	}
	return nil
}

// annotate runs the hook for one entry and returns the text to append.
//
// The command runs *in the directory being listed*, not in sl's own working
// directory. That is what makes the useful hooks useful: `git status --
// "$1"` only means anything inside the repository, and `sl ~/project` from
// somewhere else must behave like `sl` started in ~/project. This was a bug:
// the hook inherited sl's cwd, so the git example silently annotated nothing
// unless the listed directory happened to be the current one.
func (c *Ctx) annotate(dir, path, name string) string {
	if c.opts.Annotate == "" {
		return ""
	}
	if c.annotateRe != nil && !c.annotateRe.MatchString(name) {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", c.opts.Annotate, "sh", path, name)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return ""
	}
	// keep the layout sane: no escapes, no tabs, bounded length
	return cleanAnnotation(string(out))
}

// annotateBatch runs the hook once for a whole directory. The command gets the
// paths on its standard input, one per line, and must answer with one line per
// path; the i-th line annotates the i-th path.
//
// The Tcl original called perfile() as a procedure, so it paid nothing per
// entry; --annotate pays one process per entry. This is the middle ground: one
// process per directory, and the annotations still arrive with the listing.
// The directory is passed as $1; there is no $2, because there is no single
// name.
//
// Two limits are inherent and are documented in the manual: a name containing
// a newline cannot be represented in a line protocol (such entries are left
// unannotated), and the command sees all the names at once, so it cannot
// depend on being called once per entry.
func (c *Ctx) annotateBatch(dir string, paths []string) {
	if c.opts.AnnotateBatch == "" {
		return
	}
	// only names that the match pattern accepts, and only names we can
	// actually send: the answer has to line up with the input
	want := make([]string, 0, len(paths))
	for _, p := range paths {
		name := filepathBase(p)
		if strings.ContainsAny(p, "\n") {
			continue
		}
		if c.annotateRe != nil && !c.annotateRe.MatchString(name) {
			continue
		}
		want = append(want, p)
	}
	if len(want) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), annotateBatchTimeout)
	defer cancel()

	// $1 is the directory being listed, so a command can use it without
	// having to derive it from the paths (which may be relative)
	cmd := exec.CommandContext(ctx, "sh", "-c", c.opts.AnnotateBatch, "sh", dir)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(strings.Join(want, "\n") + "\n")
	var errb strings.Builder
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		fmt.Fprintf(c.errw, "sl: annotate-batch: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if len(lines) != len(want) {
		// A mismatch means the command broke the protocol (it printed a
		// header, a warning, or one answer for several paths). Answers are
		// positional, so keeping them would attach each one to the wrong
		// name: say so once and show no annotations at all.
		fmt.Fprintf(c.errw, "sl: annotate-batch: %d lines for %d paths, annotations dropped\n",
			len(lines), len(want))
		return
	}
	c.batchAnn = make(map[string]string, len(want))
	for i, p := range want {
		if a := cleanAnnotation(lines[i]); a != "" {
			c.batchAnn[p] = " " + a
		}
	}
}

// cleanAnnotation applies the same rules as the per-entry hook: one short line,
// no control characters, so a stray "fatal: not a git repository" cannot wreck
// the layout.
func cleanAnnotation(line string) string {
	line = strings.TrimSpace(strings.SplitN(line, "\n", 2)[0])
	if line == "" {
		return ""
	}
	line = strings.Map(func(r rune) rune {
		if r < 32 || r == 0x7f {
			return ' '
		}
		return r
	}, line)
	if r := []rune(line); len(r) > 24 {
		line = string(r[:24])
	}
	return line
}

// filepathBase is filepath.Base without pulling the import into the hot path of
// this file; it keeps the batch code readable next to the per-entry hook.
func filepathBase(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}
