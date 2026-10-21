package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// TestInitConfigCreatesOnce: the first run writes the template, the second one
// leaves the file alone (a user's edits must never be overwritten).
func TestInitConfigCreatesOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sl", "config")

	got, err := ensureUserConfig(path, true)
	if err != nil {
		t.Fatalf("ensureUserConfig: %v", err)
	}
	if got != path {
		t.Fatalf("created %q, want %q", got, path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat: %v", err)
	}

	if err := os.WriteFile(path, []byte("# mine\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got, err := ensureUserConfig(path, true); err != nil || got != "" {
		t.Fatalf("second call = (%q, %v), want (\"\", nil)", got, err)
	}
	if b, _ := os.ReadFile(path); string(b) != "# mine\n" {
		t.Errorf("existing config was overwritten: %q", b)
	}
}

// TestInitConfigTemplateIsNeutral is the important one: every setting in the
// template is commented out, so parsing a freshly created config must leave the
// built-in defaults untouched. If a default changes without the template
// following, this test says so.
func TestInitConfigTemplateIsNeutral(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sl", "config")
	if _, err := ensureUserConfig(path, true); err != nil {
		t.Fatal(err)
	}

	want := defaultOptions()
	got := defaultOptions()
	if err := loadConfig(path, true, got); err != nil {
		t.Fatalf("the generated template does not parse: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("generated config changes the defaults:\n got %+v\nwant %+v", got, want)
	}
}

// TestInitConfigEveryKeyWorks checks that each active "key = value" line in the
// template is accepted by the config parser -- the comments next to the keys
// are meant to be uncommented by hand, and a typo there would be invisible.
func TestInitConfigEveryKeyWorks(t *testing.T) {
	// group.*.extra and style.* write into package-level tables, which a test
	// binary shares with every other test; put them back afterwards.
	defer saveTables()()

	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	// uncomment everything, keeping the values: if the parser rejects a key or
	// a value, this fails and points at the line.
	var out []byte
	for _, line := range splitLines(configTemplate) {
		if body, ok := commentedSetting(line); ok {
			out = append(out, body...)
			out = append(out, '\n')
			continue
		}
		out = append(out, line...)
		out = append(out, '\n')
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
	o := defaultOptions()
	if err := loadConfig(path, true, o); err != nil {
		t.Fatalf("uncommented template does not parse: %v", err)
	}
}

// saveTables snapshots the mutable tables the config parser can patch and
// returns a function that restores them.
func saveTables() func() {
	eqCopy := make(map[string]string, len(eq))
	for k, v := range eq {
		eqCopy[k] = v
	}
	styleCopy := make(map[string]string, len(styleMap))
	for k, v := range styleMap {
		styleCopy[k] = v
	}
	return func() {
		eq = eqCopy
		styleMap = styleCopy
	}
}

// commentedSetting returns the body of a "# key = value" line, and false for
// prose comments such as "# ---- what to show".
func commentedSetting(line string) (string, bool) {
	if len(line) < 4 || line[0] != '#' || line[1] != ' ' {
		return "", false
	}
	body := line[2:]
	eq := strings.Index(body, " = ")
	if eq <= 0 {
		return "", false
	}
	for _, r := range body[:eq] {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '.' && r != '-' {
			return "", false
		}
	}
	return body, true
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// TestConfigTemplateShowsRealDefaults checks the promise the template makes:
// every commented value IS the built-in default, so that uncommenting a line
// never changes anything by surprise.
//
// This test exists because of a bug a user hit: the template said
//
//	# one = on            # one entry per line
//
// while the default is multi-column. The old test only checked that the file as
// written changes nothing -- which is true, because every line is commented
// out. A misleading comment is invisible to that check.
func TestConfigTemplateShowsRealDefaults(t *testing.T) {
	want := defaultOptions()
	got := defaultOptions()

	// Lines under these headings are documented examples (git log, group
	// extras), not defaults; the template says so in the text above them.
	examples := map[string]bool{
		"annotate": true, "annotate-batch": true, "annotate-match": true, "rule": true,
	}

	for _, line := range strings.Split(configTemplate, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "# ") {
			continue
		}
		line = strings.TrimSpace(line[2:])
		eq := strings.Index(line, " = ")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := stripInlineComment(line[eq+3:])
		if !configKeySyntax.MatchString(key) || val == "" {
			continue
		}
		if examples[key] || strings.HasPrefix(key, "style") || strings.HasPrefix(key, "group.") {
			continue
		}
		if err := applyConfigKey(key, val, got); err != nil {
			t.Errorf("template line %q does not parse: %v", line, err)
		}
	}

	if diffs := diffOptions(want, got); len(diffs) > 0 {
		t.Errorf("the template documents values that are not the defaults:\n  %s",
			strings.Join(diffs, "\n  "))
	}
}

// configKeySyntax accepts "ignore", "one", "owner-notes", "group.doc.extra";
// it rejects prose that happens to contain " = " (the header comment does).
var configKeySyntax = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9.-]*$`)

// diffOptions names the fields in which two option sets differ, so that a
// failure says "one = on, want off" rather than printing two structs.
func diffOptions(want, got *Options) []string {
	var out []string
	v, g := reflect.ValueOf(*want), reflect.ValueOf(*got)
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if !reflect.DeepEqual(v.Field(i).Interface(), g.Field(i).Interface()) {
			out = append(out, fmt.Sprintf("%s: template %v, default %v",
				typ.Field(i).Name, g.Field(i).Interface(), v.Field(i).Interface()))
		}
	}
	return out
}
