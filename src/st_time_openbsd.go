//go:build openbsd

package main

// OpenBSD names the timestamp fields Mtim/Atim; see st_time_bsd.go.

import "syscall"

func stMtime(st *syscall.Stat_t) syscall.Timespec { return st.Mtim }
func stAtime(st *syscall.Stat_t) syscall.Timespec { return st.Atim }
