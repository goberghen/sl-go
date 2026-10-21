//go:build darwin || freebsd || netbsd

package main

// The timestamp fields of syscall.Stat_t differ across the BSDs: FreeBSD,
// NetBSD and Darwin call them Mtimespec/Atimespec, OpenBSD calls them
// Mtim/Atim (see st_time_openbsd.go). Keeping the difference in two four-line
// files is cheaper than a build-tag maze in statOf, and it makes the day the
// names change again a two-line change.

import "syscall"

func stMtime(st *syscall.Stat_t) syscall.Timespec { return st.Mtimespec }
func stAtime(st *syscall.Stat_t) syscall.Timespec { return st.Atimespec }
