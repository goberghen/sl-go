package main

import "time"

// Entry replaces the 13-element Tcl list of the original ("... could use a struct
// here", said the comment in the script). Every stage of the pipeline reads and
// writes fields of this struct: collect -> mark up -> sort -> lay out -> render.
type Entry struct {
	Tail    string // raw name on disk (what you'd type)
	Sfx     string // lowercased suffix, "" for dirs
	Type    string // "file", "directory", "link", "special" or "GROUP" for a title
	Group   string
	Sortkey string
	Style   string // concatenated SGR style codes, e.g. ";7;31"
	Pre     string // 1-3 char marker printed before the name
	Display string
	Warning string
	Post    string
	Size    int64
	Mtime   time.Time
	Atime   time.Time

	// filled in by the layout code
	Len  int // full width of the cell
	Lenx int // width clamped to the column budget
}

func (e *Entry) isDir() bool    { return e.Type == "directory" }
func (e *Entry) isTitle() bool  { return e.Type == "GROUP" }
func (e *Entry) clone() *Entry  { c := *e; return &c }
func (e *Entry) atimeSet() bool { return !e.Atime.IsZero() }
