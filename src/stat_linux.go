//go:build linux

package main

import (
	"os"
	"syscall"
	"time"
)

func statOf(fi os.FileInfo) fileStat {
	var s fileStat
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		s.mode = uint32(st.Mode)
		s.uid = st.Uid
		s.nlink = uint64(st.Nlink)
		s.size = st.Size
		s.mtime = time.Unix(int64(st.Mtim.Sec), int64(st.Mtim.Nsec))
		s.atime = time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec))
		s.hasAcc = true
		return s
	}
	s.size = fi.Size()
	s.mtime = fi.ModTime()
	s.atime = fi.ModTime()
	return s
}
