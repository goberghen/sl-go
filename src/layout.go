package main

// Column layout and rendering.
// Port of proc column / tailor / bump, with the same "try a shape, keep the one
// with the fewest group titles at the bottom" heuristic.

import (
	"strings"
)

// render lays out and prints the entries.
func (c *Ctx) render(list []*Entry, gtitcnt int) {
	llen := len(list)
	if llen == 0 {
		return
	}
	width := c.width - 1 // right margin (the left one comes free with pre)
	if c.opts.One {
		width = 1
	}

	// 1. column budget
	if c.opts.NameMin <= 0 {
		c.opts.NameMin = 40
	}
	colsmin := width / c.opts.NameMin
	namemax := 1000000
	if colsmin > 1 && c.opts.Shorten > 0 && llen >= c.opts.Shorten {
		namemax = (width - c.colSep()*(colsmin-1)) / colsmin
	}

	maxlen := 0
	for _, e := range list {
		e.Len = displayLen(e.Display) + len(e.Pre) + displayLen(e.Post) + len(e.Warning)
		e.Lenx = e.Len
		if e.Lenx > namemax {
			e.Lenx = namemax
		}
		if e.Len > maxlen {
			maxlen = e.Len
		}
	}
	if maxlen > namemax {
		c.logf("|cell| %d->%d", maxlen, namemax)
	}
	maxw := maxlen
	if maxw > namemax {
		maxw = namemax
	}
	colsuni := width / (maxw + c.colSep())
	if colsuni == 0 {
		colsuni = 1
	}
	c.logf("uni %d/%d=%dx%d", width, maxw, colsuni, iceil(llen, colsuni))

	// 2a. as many columns as fit
	rows := llen
	for cols := colsuni; ; {
		r := iceil(llen, cols)
		if r == 0 {
			break
		}
		sumx, _, _ := c.tailor(list, 0, llen, r)
		if sumx > width {
			c.logf("X%dx%d = %d", cols, r, sumx)
			break
		}
		rows = r
		if r == 1 {
			break
		}
		cols = iceil(llen, r-1)
	}

	// 2b. aesthetic refinements
	if rows == 1 && gtitcnt >= 1 {
		rows++
	}
	ll1, bot1 := c.bump(list, rows, width)
	cols1 := iceil(len(ll1), rows)
	c.logf("tailored %dx%d bot=%d", cols1, rows, bot1)
	var ll []*Entry
	if c.opts.MaxCol || rows == 1 || cols1 == 1 {
		ll = ll1
	} else {
		rows2 := rows + 1
		ll2, bot2 := c.bump(list, rows2, width)
		cols2 := iceil(len(ll2), rows2)
		switch {
		case bot2 < bot1:
			ll, rows = ll2, rows2
			c.logf("row+1 %dx%d (< bot=%d->%d)", cols2, rows2, bot1, bot2)
		case bot2 == bot1 && cols2+1 == cols1 && cols2 >= 4:
			ll, rows = ll2, rows2
			c.logf("row+1 %dx%d (< cols)", cols2, rows2)
		default:
			ll = ll1
		}
	}

	// 3. widths, then hand out the slack to the columns that were clamped
	llen = len(ll)
	sumx, wxl, wl := c.tailor(ll, 0, llen, rows)
	cols := len(wxl)
	slack := width - sumx
	slack0 := slack
	for {
		ate := false
		for i := 0; i < cols && slack > 0; i++ {
			if wxl[i] < wl[i] {
				wxl[i]++
				slack--
				ate = true
			}
		}
		if !ate {
			break
		}
	}
	if slack < slack0 {
		c.logf("+%d widths", slack0-slack)
	}
	if gut := cols - 1; gut > 0 && slack >= gut {
		c.logf("+1*%d gutter", gut)
		for i := 0; i < gut; i++ {
			wxl[i]++
		}
	}

	// 4. print
	if c.opts.One {
		wxl = []int{namemax}
	}
	shortened := 0
	for i := 0; i < rows; i++ {
		for j, n := 0, i; n < llen; j, n = j+1, n+rows {
			e := ll[n]
			w := wxl[j]
			// a nil is a placeholder inserted by bump() to push a group
			// title out of the bottom cell; trailing placeholders must not
			// leave spaces at the end of the line
			hasnextcol := false
			for m := n + rows; m < llen; m += rows {
				if ll[m] != nil {
					hasnextcol = true
					break
				}
			}
			gap := 0
			if e == nil {
				gap = w - c.colSep()
			} else {
				c.w.WriteString(c.color(e.Pre, "pre", "OFF"))
				if e.Style != "" {
					c.w.WriteString(c.colorcode(e.Style))
				}

				availw := w
				if hasnextcol {
					availw -= c.colSep()
				} else if j+1 < cols {
					availw += wxl[j+1] // spill into the next column
				}

				var show string
				if availw >= e.Len {
					show = e.Display
					gap = availw - e.Len
				} else {
					availw -= len(e.Pre) + displayLen(e.Post) + len(e.Warning)
					show = c.shorten(e.Display, availw)
					gap = availw - displayLen(show)
					shortened++
				}

				// autosearch is applied last, so shortening can't hide the match
				if loc := c.autosearchLoc(show); loc != nil {
					a := show[:loc[0]]
					b := show[loc[0]:loc[1]]
					rest := show[loc[1]:]
					show = a + c.color(b, "autosearch", e.Style) + rest
				}

				c.w.WriteString(show)
				if e.Style != "" {
					c.w.WriteString(c.colorcode("OFF"))
				}
				if e.Warning != "" {
					c.w.WriteString(c.color(e.Warning, "warning", "OFF"))
				}
				c.w.WriteString(c.color(e.Post, "post", "OFF"))
			}
			if hasnextcol {
				c.w.WriteString(spaces(gap + c.colSep()))
			}
		}
		c.w.WriteString("\n")
	}
	if shortened > 0 {
		c.logf("%d names shortened", shortened)
	}
}

// tailor returns the total width of a shape with `rows` rows, the clamped column
// widths (wxl) and the unclamped ones (wl).
func (c *Ctx) tailor(l []*Entry, i, llen, rows int) (sumx int, wxl, wl []int) {
	c.logf("tailor")
	for ; i < llen; i += rows {
		max, maxx := 0, 0
		for j := i; j < i+rows && j < llen; j++ {
			if l[j] == nil {
				continue
			}
			if l[j].Len > max {
				max = l[j].Len
			}
			if l[j].Lenx > maxx {
				maxx = l[j].Lenx
			}
		}
		if i+rows < llen {
			max += c.colSep()
			maxx += c.colSep()
		}
		sumx += maxx
		wl = append(wl, max)
		wxl = append(wxl, maxx)
	}
	return sumx, wxl, wl
}

// bump pushes a group title that landed at the bottom of a column into the next
// column, if the widths still fit.
func (c *Ctx) bump(ll []*Entry, rows, width int) ([]*Entry, int) {
	llen := len(ll)
	bot := 0
	if rows > 1 {
		for i := rows - 1; i < llen; i += rows {
			if ll[i] != nil && ll[i].isTitle() {
				bot++
			}
		}
	}
	if bot == 0 {
		return ll, 0
	}

	_, wxl, _ := c.tailor(ll, 0, llen, rows)
	var out []*Entry
	botbump := 0
	left, col, row := 0, 0, 0
	for i := 0; i < llen; i++ {
		e := ll[i]
		if row == 0 && col < len(wxl) {
			left += wxl[col]
		}
		if row+1 == rows && e != nil && e.isTitle() {
			// would bumping this title into the next column still fit?
			s2, wxl2, _ := c.tailor(ll, i, llen, rows)
			if left+s2 <= width {
				out = append(out, nil)
				wxl = wxl2
				col, row = 0, 0
				left += wxl[0]
			} else {
				botbump++
			}
		}
		out = append(out, e)
		if row+1 < rows {
			row++
		} else {
			col++
			row = 0
		}
	}
	if botbump < bot {
		return out, botbump
	}
	return ll, bot
}

// colSep is the minimum gap between columns (the original's COLSEP).
func (c *Ctx) colSep() int {
	if c.opts.ColSep < 0 {
		return 0
	}
	return c.opts.ColSep
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}
