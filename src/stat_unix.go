//go:build unix

package main

import (
	"syscall"
	"time"
)

type fileStat struct {
	mode   uint32
	uid    uint32
	nlink  uint64
	size   int64
	atime  time.Time
	mtime  time.Time
	hasAcc bool
}

func isReadable(path string) bool   { return syscall.Access(path, 4) == nil }
func isExecutable(path string) bool { return syscall.Access(path, 1) == nil }

// resourceForkLen: on macOS a resource fork can carry the size of an otherwise
// empty file. Not applicable elsewhere.
func resourceForkLen(path string) int64 { return 0 }
