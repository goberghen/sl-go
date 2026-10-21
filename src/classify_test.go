package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestClassifyByType guards the fix for the "no suffix means directory"
// heuristic: a directory full of suffix-less FILES must not look like a
// hierarchy, while a directory that really holds subdirectories must.
func TestClassifyByType(t *testing.T) {
	dir := t.TempDir()

	// four suffix-less regular files and eight images
	for _, n := range []string{"aaa", "bbb", "ccc", "ddd"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"1.jpg", "2.jpg", "3.jpg", "4.jpg", "5.jpg", "6.jpg", "7.jpg", "8.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	content, ok := dirContent(dir)
	if !ok {
		t.Fatal("dirContent failed")
	}
	if g, size, _ := classifyDir(dir, "photos", content, false); g != "image" {
		t.Errorf("new rule: dir of suffix-less files classified as %q, want image", g)
	} else if size != 12 {
		t.Errorf("size = %d, want 12", size)
	}
	if g, _, _ := classifyDir(dir, "photos", content, true); g != "dir" {
		t.Errorf("--legacy-classify: got %q, want dir (the Tcl heuristic)", g)
	}

	// the same names as real subdirectories: a hierarchy again
	dir2 := t.TempDir()
	for _, n := range []string{"aaa", "bbb", "ccc"} {
		if err := os.MkdirAll(filepath.Join(dir2, n), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"1.mp3", "2.mp3", "3.mp3", "4.mp3", "5.mp3", "6.mp3", "7.mp3"} {
		if err := os.WriteFile(filepath.Join(dir2, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	content2, _ := dirContent(dir2)
	if g, _, _ := classifyDir(dir2, "music", content2, false); g != "dir" {
		t.Errorf("three real subdirectories: got %q, want dir", g)
	}
}

// TestEntryIsDir covers the d_type cases, including a symlink to a directory
// and a name without a suffix (where DT_UNKNOWN needs an lstat fallback).
func TestEntryIsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plain"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub", filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}

	content, ok := dirContent(dir)
	if !ok {
		t.Fatal("dirContent failed")
	}
	got := map[string]bool{}
	for _, c := range content {
		got[c.name] = c.isDir
	}
	want := map[string]bool{"sub": true, "plain": false, "file.txt": false, "link": true, "link.txt": false}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("entryIsDir(%q) = %v, want %v", name, got[name], w)
		}
	}
}

// TestUserRules: a rule forces a group and beats the built-in tables.
func TestUserRules(t *testing.T) {
	if err := compileRules([]string{
		`^IMG_[0-9]+\.JPG$ -> image`,
		`\.tmp$ -> ignore`,
		`^notes$ -> dev`,
	}); err != nil {
		t.Fatalf("compileRules: %v", err)
	}
	defer func() { userRules = nil }()

	if g, _, _, _, _ := classify("/x", "IMG_42.JPG", "file"); g != "image" {
		t.Errorf("IMG_42.JPG: group %q, want image", g)
	}
	if g, ig, _, _, _ := classify("/x", "leftover.tmp", "file"); g != "ignore" || ig != 1 {
		t.Errorf("leftover.tmp: (%q, ig=%d), want (ignore, 1)", g, ig)
	}
	// the rule sees the name with the wrapper suffix stripped
	if g, _, _, _, _ := classify("/x", "notes.gz", "file"); g != "dev" {
		t.Errorf("notes.gz: group %q, want dev (rule must see \"notes\")", g)
	}
	// no match: the built-in table decides
	if g, _, _, _, _ := classify("/x", "song.mp3", "file"); g != "audio" {
		t.Errorf("song.mp3: group %q, want audio", g)
	}
	// first match wins, later rules do not override it
	if err := compileRules([]string{`.log$ -> dev`, `.log$ -> doc`}); err != nil {
		t.Fatal(err)
	}
	if g, _, _, _, _ := classify("/x", "x.log", "file"); g != "dev" {
		t.Errorf("x.log: group %q, want dev (first rule wins)", g)
	}
	userRules = nil
	if err := compileRules([]string{`.log$ -> nosuchgroup`}); err == nil {
		t.Error("an unknown group must be an error")
	}
	if err := compileRules([]string{`^\.log$`}); err == nil {
		t.Error("a rule without \"-> group\" must be an error")
	}
	if err := compileRules([]string{`[ -> dev`}); err == nil {
		t.Error("a broken regexp must be an error")
	}
	userRules = nil
}

// TestColourOption: --color[=WHEN] and the config key accept the three words,
// and a bare -c still means "always".
func TestColourOption(t *testing.T) {
	cases := []struct {
		arg  string
		want int
	}{
		{"--color", 1}, {"--color=always", 1}, {"--color=auto", -1},
		{"--color=never", 0}, {"--color=off", 0}, {"-c", 1},
		{"--no-color", 0},
	}
	for _, tc := range cases {
		o, _, err := parseArgs([]string{tc.arg})
		if err != nil {
			t.Errorf("%s: %v", tc.arg, err)
			continue
		}
		if o.Color != tc.want {
			t.Errorf("%s: Color = %d, want %d", tc.arg, o.Color, tc.want)
		}
	}
	if _, _, err := parseArgs([]string{"--color=sometimes"}); err == nil {
		t.Error("--color=sometimes must be an error")
	}
	// the config key goes through the same code
	o := defaultOptions()
	o.Color = 1
	if err := applyConfigKey("color", "auto", o); err != nil || o.Color != -1 {
		t.Errorf("color = auto: Color = %d, err = %v", o.Color, err)
	}
	if err := applyConfigKey("one", "off", defaultOptions()); err != nil {
		t.Errorf("one = off: %v", err)
	}
	o = defaultOptions()
	o.One = true
	if err := applyConfigKey("one", "off", o); err != nil || o.One {
		t.Errorf("one = off did not switch one-column mode off (One = %v)", o.One)
	}
	o = defaultOptions()
	if err := applyConfigKey("k1000", "off", o); err != nil || o.Base != 1024 {
		t.Errorf("k1000 = off: Base = %d, err = %v", o.Base, err)
	}
}
