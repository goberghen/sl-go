// Command completions writes the shell completion files from the option table.
//
//	go run ./tools/completions -src src/cli.go -out .
//
// Why a separate program and not a flag of sl itself: completion files are
// static data that belongs in the package, not something a 2 MB utility should
// print on demand. `make completions` runs this once, the result is committed,
// and CI-style test TestCompletionsMatchOptions fails if the committed files
// and the option table ever drift apart -- which is the failure mode that
// matters, because a completion file that offers a removed option is worse than
// none at all.
//
// The option table is read here with a regular expression rather than by
// importing the package (package main cannot be imported). That is safe only
// because the test compares the generated output with the real table: if the
// regex misses a field, the test says so.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// opt is one entry of the table in src/cli.go.
type opt struct {
	long   string
	short  string // "" when there is no short form
	hasArg bool   // value required, may be a separate word
	optArg bool   // value optional, attached with "="
	arg    string // placeholder: N, RE, CMD, WHEN, ...
	help   string // one-line description, shown by zsh and fish
}

var reLong = regexp.MustCompile(`long:\s*"([^"]+)"`)

// reHelp grabs the body of the help: "..." literal; it stops at the first
// unescaped quote, and strconv.Unquote turns the escapes back into text.
var reHelp = regexp.MustCompile(`help:\s*"((?:[^"\\]|\\.)*)"`)

func main() {
	src := flag.String("src", "src/cli.go", "the file holding the option table")
	out := flag.String("out", "completions", "output directory")
	flag.Parse()

	opts, err := readTable(*src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "completions: %v\n", err)
		os.Exit(1)
	}
	if len(opts) == 0 {
		fmt.Fprintln(os.Stderr, "completions: no options found; the table format changed")
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "completions: %v\n", err)
		os.Exit(1)
	}

	files := []struct {
		name string
		body string
	}{
		{"sl.bash", bash(opts)},
		{"_sl", zsh(opts)},
		{"sl.fish", fish(opts)},
	}
	for _, f := range files {
		path := filepath.Join(*out, f.name)
		if err := os.WriteFile(path, []byte(f.body), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "completions: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
}

// readTable pulls the option entries out of src/cli.go. Every entry has the
// shape {long: "...", short: 'x', hasArg: true, arg: "N", ...}, spread over one
// or two lines, so splitting the source on "\t\t{long:" is both simple and
// exact enough -- the test verifies the result against the real table.
func readTable(path string) ([]opt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), "{long:")
	var out []opt
	for _, p := range parts[1:] {
		head := p
		if i := strings.IndexByte(head, '}'); i >= 0 {
			head = head[:i+1]
		}
		m := reLong.FindStringSubmatch("{long:" + head)
		if m == nil {
			continue
		}
		o := opt{long: m[1]}
		if s := regexp.MustCompile(`short:\s*'([^']*)'`).FindStringSubmatch(head); s != nil {
			o.short = s[1]
		}
		if strings.Contains(head, "hasArg: true") {
			o.hasArg = true
		}
		if strings.Contains(head, "optArg: true") {
			o.optArg = true
		}
		if a := regexp.MustCompile(`arg:\s*"([^"]*)"`).FindStringSubmatch(head); a != nil {
			o.arg = a[1]
		}
		// The help text is the description every shell shows next to the
		// option, so it has to survive the trip unchanged: it is a Go string
		// literal in cli.go, so unquote it rather than regexp-ing the escapes.
		if h := reHelp.FindStringSubmatch(head); h != nil {
			if v, err := strconv.Unquote(`"` + h[1] + `"`); err == nil {
				o.help = v
			} else {
				o.help = h[1]
			}
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].long < out[j].long })
	return out, nil
}

// valueCandidates lists the fixed values some options accept; with these the
// shell offers real choices instead of falling back to file names.
var valueCandidates = map[string][]string{
	"color":  {"auto", "always", "never"},
	"k":      {"1000", "1024"},
	"ignore": {"0", "1", "2", "3"},
}

// groups are the names a --rule may assign. They come from names.go
// (groupDefs plus the names that exist without a table entry); the test checks
// this list against the program's own groupNames().
var groups = []string{
	"special", "dir", "build", "app", "dev", "A2", "WWW", "aux", "doc", "book",
	"document", "image", "vector", "audio", "video", "font", "data", "other",
	"archive", "dotfile", "Mac", "file", "ignore",
}

func longs(opts []opt) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.long)
	}
	return out
}

// withValue are the options after which a value is expected (the "--opt value"
// form); bash needs them to fall through to file completion or to a fixed list.
func classify(opts []opt) (flags, valued []string, optional []string, fixed map[string][]string) {
	fixed = map[string][]string{}
	for _, o := range opts {
		switch {
		case o.optArg:
			optional = append(optional, o.long)
			if v, ok := valueCandidates[o.long]; ok {
				fixed[o.long] = v
			}
		case o.hasArg:
			valued = append(valued, o.long)
			if v, ok := valueCandidates[o.long]; ok {
				fixed[o.long] = v
			}
		default:
			flags = append(flags, o.long)
		}
	}
	return flags, valued, optional, fixed
}
