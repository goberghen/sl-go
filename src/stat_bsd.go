//go:build darwin || freebsd || netbsd || openbsd

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
		s.uid = uint32(st.Uid)
		s.nlink = uint64(st.Nlink)
		s.size = st.Size
		// The fields are int32 on 32-bit targets (i386, 32-bit ARM) and
		// int64 elsewhere; time.Unix wants int64 either way.
		s.mtime = time.Unix(int64(stMtime(st).Sec), int64(stMtime(st).Nsec))
		s.atime = time.Unix(int64(stAtime(st).Sec), int64(stAtime(st).Nsec))
		s.hasAcc = true
		return s
	}
	s.size = fi.Size()
	s.mtime = fi.ModTime()
	s.atime = fi.ModTime()
	return s
}
