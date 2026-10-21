package main

// Statistical "relative" marking: instead of hard thresholds ("modified in the
// last hour") the original compares each entry with the rest of the listing and
// marks the top few percent by size, by mtime and by atime.
// Port of proc relative / proc cutoff.

import "time"

func (c *Ctx) relative(list []*Entry) []*Entry {
	abscutoff := c.now.Unix() - c.opts.AbsChange

	var sizes, mtimes, atimes []int64
	for _, e := range list {
		sizes = append(sizes, e.Size)
		mtimes = append(mtimes, e.Mtime.Unix())
		if !e.isDir() {
			atimes = append(atimes, e.Atime.Unix())
		}
	}

	cutoffSize, repz := c.cutoff(sizes, c.opts.RelSize, 1)
	cutoffM, repm := c.cutoff(mtimes, c.opts.RelChange, 5*secMinute)
	cutoffA, repa := c.cutoff(atimes, c.opts.RelRead, 5*secMinute)
	c.logf("NOW = %d", c.now.Unix())

	out := make([]*Entry, 0, len(list))
	for _, e0 := range list {
		e := e0.clone()
		sub := "file"
		if e.isDir() {
			sub = "dir"
		}

		if c.opts.RelSize > 0 && (e.Size > cutoffSize || (e.Size == cutoffSize && repz)) {
			e.Pre = "   "
			if !e.isDir() {
				e.Post += " " + c.kmg(e.Size, "post")
			}
			repz = false
		}

		if c.opts.AbsChange > 0 && e.Mtime.Unix() >= abscutoff {
			e.Style += styleMap["recent,"+sub]
			if e.Mtime.Unix() > cutoffM || c.now.Unix()-e.Mtime.Unix() < secHour {
				e.Post += " <" + c.ago(e.Mtime)
				repm = false
			}
		} else if c.opts.RelChange > 0 && (e.Mtime.Unix() > cutoffM || (e.Mtime.Unix() == cutoffM && repm)) {
			e.Style += styleMap["relrec,"+sub]
			e.Post += " <" + c.ago(e.Mtime)
			repm = false
		}

		if !e.isDir() {
			switch {
			case c.opts.Atime > 0:
				if c.now.Unix()-e.Atime.Unix() <= c.opts.Atime {
					e.Pre = "->" + drop2(e.Pre)
					e.Post += " -" + c.ago(e.Atime)
					repa = false
				}
			case c.opts.RelRead > 0 &&
				(e.Atime.Unix() > cutoffA || (e.Atime.Unix() == cutoffA && repa)) &&
				e.Atime.Unix() > e.Mtime.Unix()+secMinute:
				e.Pre = "->" + drop2(e.Pre)
				e.Post += " -" + c.ago(e.Atime)
				repa = false
			}
		}
		out = append(out, e)
	}
	return out
}

// cutoff returns the value that separates "interesting" entries from the rest,
// plus whether one entry should be marked even if nothing stands out.
func (c *Ctx) cutoff(vals []int64, pct int, by int64) (int64, bool) {
	if len(vals) == 0 || pct == 0 {
		return 0, false
	}
	if len(vals) == 1 {
		return vals[0], true
	}

	s := append([]int64(nil), vals...)
	sortDescending(s)
	max := s[0]

	cut := (len(s)*pct)/100 + 1
	if cut > pct {
		cut = pct
	}
	if cut >= len(s) {
		cut = len(s) - 1
	}
	val := s[cut]
	rep := false

	if len(s) > 4 {
		// values have to differ by a meaningful amount, otherwise "all
		// downloaded a minute apart a year ago" would look interesting
		beat := s[len(s)/2] + by
		if val < beat {
			val = beat
		}
	}
	if max <= val {
		val = max
		if !c.opts.Only {
			rep = true
			c.logf("rep=%s", what(pct))
		}
	}
	return val, rep
}

func what(pct int) string {
	switch pct {
	case 5:
		return "relsize"
	case 10:
		return "relchange/relread"
	}
	return "cutoff"
}

func sortDescending(s []int64) {
	for i := 1; i < len(s); i++ {
		v := s[i]
		j := i - 1
		for ; j >= 0 && s[j] < v; j-- {
			s[j+1] = s[j]
		}
		s[j+1] = v
	}
}

func drop2(s string) string {
	r := []rune(s)
	if len(r) < 2 {
		return ""
	}
	return string(r[2:])
}

var _ = time.Now
