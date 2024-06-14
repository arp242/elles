//go:build freebsd || netbsd || darwin

package os2

import (
	"io/fs"
	"syscall"
	"time"
)

func Atime(fi fs.FileInfo) time.Time {
	if fi.Sys() == nil {
		return time.Time{}
	}
	t := fi.Sys().(*syscall.Stat_t).Atimespec
	return time.Unix(int64(t.Sec), int64(t.Nsec))
}

func Btime(absdir string, fi fs.FileInfo) time.Time {
	if fi.Sys() == nil {
		return time.Time{}
	}
	t := fi.Sys().(*syscall.Stat_t).Birthtimespec
	return time.Unix(int64(t.Sec), int64(t.Nsec))
}
