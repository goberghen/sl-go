package main

// Display helpers: SGR styles, cell width, human sizes, "ago", permissions,
// and the middle-ellipsis shortener.

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var styleMap = map[string]string{
	"group":       ";01;35;4",
	"file":        "",
	"dir":         ";34",
	"special":     ";36",
	"link":        ";35",
	"notable":     ";7",
	"autosearch":  ";36",
	"recent,file": ";31",
	"recent,dir":  ";31",
	"relrec,file": ";33",
	"relrec,dir":  ";33",
	"relsize":     ";36",
	"warning":     ";1;01;31",
	"pre":         ";36",
	"post":        "",
	"log":         ";4",
	"OFF":         ";0",
}

var sgrOnlyRe = regexp.MustCompile(`^[;0-9]+$`)

// colorcode turns a style name (or a literal ";1;31" string) into an escape.
func (c *Ctx) colorcode(style string) string {
	if !c.colorOn || style == "" {
		return ""
	}
	if !sgrOnlyRe.MatchString(style) {
		style = styleMap[style]
	}
	if style == "" {
		return ""
	}
	return "\x1b[" + style[1:] + "m"
}

// color wraps txt in style and restores the surrounding context style after it.
func (c *Ctx) color(txt, style, context string) string {
	if txt == "" || strings.TrimSpace(txt) == "" {
		return txt
	}
	on := c.colorcode(style)
	if on == "" {
		return txt
	}
	off := ""
	if context != "OFF" {
		off = c.colorcode("OFF")
	}
	return on + txt + off + c.colorcode(context)
}

// ---------------------------------------------------------------- cell width

// displayLen is the number of terminal columns in s: escape sequences,
// combining accents and (since the Go port) wide CJK/emoji characters are
// handled correctly. The Tcl original counted one column per character.
func displayLen(s string) int { return stringWidth(s) }

// ---------------------------------------------------------------- humanize

// kmg formats a size like "4.2M"; the unit letter is tinted so that the number
// stays readable in a styled cell.
func (c *Ctx) kmg(size int64, context string) string {
	units := "BKMGTPEZY"
	kilo := float64(c.opts.Base)
	if float64(size) < kilo {
		return fmt.Sprintf("%dB", size)
	}
	po2 := 1.0
	for i := 0; i < 5; i++ {
		next := po2 * kilo
		if float64(size) < next {
			d := float64(size) / po2
			var pretty string
			if d < 10 {
				pretty = fmt.Sprintf("%.1f", d)
			} else {
				pretty = fmt.Sprintf("%.0f", d)
			}
			pretty = strings.TrimSuffix(pretty, ".0")
			return strings.TrimSpace(pretty + c.color(string(units[i]), "relsize", context))
		}
		po2 = next
	}
	return "(too big)"
}

// ago renders an age the way the original did: 12s, 5m, 3h, 9d, 4y, or a year.
func (c *Ctx) ago(t time.Time) string {
	age := int64(c.now.Sub(t) / time.Second)
	var s, u string
	switch {
	case abs64(age) < 10:
		return "NOW"
	case age < 0:
		return "FUTURE"
	case age < 2*secMinute:
		s, u = itoa(age), "second"
	case age < 2*secHour:
		s, u = itoa(age/secMinute), "minute"
	case age < 2*secDay:
		s, u = itoa(age/secHour), "hour"
	case age < secYear:
		s, u = itoa(age/secDay), "day"
	case age < 3*secYear:
		s, u = itoa((age+2*secMonth)/secYear), "year"
	default:
		return t.Format("2006")
	}
	return s + u[:1]
}

func plural(n int, singular, pluralWord, pfx string) string {
	switch {
	case n <= 0:
		return "---"
	case n == 1:
		return itoa64(int64(n)) + " " + pfx + singular
	default:
		return itoa64(int64(n)) + " " + pfx + pluralWord
	}
}

// perm renders "rwxr-xr-x".
func perm(mode uint32) string {
	var b strings.Builder
	for s := 6; s >= 0; s -= 3 {
		m := (mode >> uint(s)) & 7
		b.WriteByte(bit(m&4, 'r'))
		b.WriteByte(bit(m&2, 'w'))
		b.WriteByte(bit(m&1, 'x'))
	}
	return b.String()
}

func bit(v uint32, ch byte) byte {
	if v != 0 {
		return ch
	}
	return '-'
}

// peculiar flags odd permission bits (e.g. g+w without u+w, or r without x on a
// directory) and returns a highlighted rwx string, or "" if nothing is odd.
func (c *Ctx) peculiar(mode uint32, etype string) string {
	if mode&0777 == 0 {
		return c.color("---------", "warning", "post")
	}
	var h uint32
	powner := (mode >> 6) & 7
	pgroup := (mode >> 3) & 7
	pothers := mode & 7

	if powner&4 == 0 {
		h |= 4 << 6
	}
	if pgroup >= 1 && pgroup <= 3 {
		h |= 4 << 3
	}
	if pothers >= 1 && pothers <= 3 {
		h |= 4
	}
	if p := powner & pgroup; p != pgroup {
		h |= (pgroup - p) << 6
	}
	if p := powner & pothers; p != pothers {
		h |= (pothers - p) << 6
	}
	if p := pgroup & pothers; p != pothers {
		h |= (pothers - p) << 3
	}
	if etype == "directory" {
		if powner&6 != 0 && powner&1 == 0 {
			h |= 1 << 6
		}
		if pgroup&6 != 0 && pgroup&1 == 0 {
			h |= 1 << 3
		}
		if pothers&6 != 0 && pothers&1 == 0 {
			h |= 1
		}
	}
	if h == 0 {
		return ""
	}
	return " " + permColored(c, perm(mode), h, 0)
}

func permColored(c *Ctx, p string, h, _ uint32) string {
	var b strings.Builder
	for bit := 8; bit >= 0; bit-- {
		ch := p[8-bit]
		if (1<<uint(bit))&h != 0 {
			b.WriteString(c.color(string(ch), "warning", "post"))
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// ---------------------------------------------------------------- shortening

var (
	articlesRe = regexp.MustCompile(`(?i)(^|\W)(a|an|the)\W`)
	prepsRe    = regexp.MustCompile(`(?i)\W(of|from|for|in|to|and|volume|vol)\W`)
	numRe      = regexp.MustCompile(`\d+`)
)

// shorten elides the middle of a name so that the head and the suffix/numbers
// stay visible: "a very long name 01.mkv" -> "a very...01.mkv".
// Widths are in terminal columns, so CJK names are cut correctly.
func (c *Ctx) shorten(txt string, w int) string {
	if w <= 0 {
		return ""
	}
	if l := displayLen(txt); l <= w {
		return txt
	}

	// throw out articles and prepositions first
	txt = articlesRe.ReplaceAllString(txt, " ")
	txt = prepsRe.ReplaceAllString(txt, "/")
	if l := displayLen(txt); l <= w {
		return txt
	}

	elen := displayLen(c.ellipsis())
	mid := c.ellipsis()
	tail := 3

	if w <= 10 {
		tail = 0
	} else if m := numRe.FindStringIndex(txt); m != nil {
		n1 := runeIndex(txt, m[0])
		n2 := runeIndex(txt, m[1]) - 1
		numlen := n2 - n1 + 1
		l := displayLen(txt)
		switch {
		case 5+1+numlen+1+tail > w:
			// too many digits to keep the number
		case n2 <= w-elen-tail-1:
			if 8+numlen+8 <= w {
				tail = 8
			}
		default:
			x := l - tail - elen - numlen - 1
			if x <= n1 {
				keep := x
				if n1 > keep {
					keep = n1
				}
				tail = l - keep
			} else {
				mid = c.ellipsis() + string([]rune(txt)[n1:n2+1]) + c.ellipsis()
				if l-tail-1 <= n2 {
					tail = l - (n2 + 1)
				}
			}
		}
	} else if w >= 20 {
		tail = 8
	}

	if tail < 0 {
		tail = 0
	}
	// a trailing space or dot carries no information: give the space to the head
	if tail > 0 {
		r := []rune(txt)
		if tail <= len(r) {
			if ch := r[len(r)-tail]; ch == ' ' || ch == '.' {
				tail--
			}
		}
	}

	headW := w - displayLen(mid) - tail
	if headW < 0 {
		headW = 0
	}
	head := truncateWidth(txt, headW)
	tailStr := truncateWidthTail(txt, tail)
	if strings.HasPrefix(txt, tailStr) && tailStr != "" {
		tailStr = "" // head already covers everything
	}
	out := head + mid + tailStr
	if displayLen(out) > w {
		out = truncateWidth(out, w)
	}
	return out
}

func (c *Ctx) ellipsis() string {
	if c.opts != nil && c.opts.Ellipsis != "" {
		return c.opts.Ellipsis
	}
	return "..."
}

// ---------------------------------------------------------------- misc

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func itoa(v int64) string   { return itoa64(v) }
func itoa64(v int64) string { return fmt.Sprintf("%d", v) }

func iceil(num, den int) int {
	if den == 0 {
		return 0
	}
	return (num + den - 1) / den
}

const (
	secMinute = 60
	secHour   = 60 * 60
	secDay    = 60 * 60 * 24
	secMonth  = 60 * 60 * 24 * 30
	secYear   = 60 * 60 * 24 * 365
)

// runeIndex maps a byte offset in s (ASCII text) to a rune offset.
func runeIndex(s string, byteOff int) int {
	n := 0
	for i := range s {
		if i >= byteOff {
			return n
		}
		n++
	}
	return n
}

var _ = unicode.IsLetter
