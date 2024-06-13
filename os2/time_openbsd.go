//go:build openbsd

package os2

import (
	"io/fs"
	"syscall"
	"time"
)

func Atime(fi fs.FileInfo) time.Time {
	t := fi.Sys().(*syscall.Stat_t).Atim
	return time.Unix(t.Sec, t.Nsec)
}

func Btime(absdir string, fi fs.FileInfo) time.Time {
	return time.Time{}
}
