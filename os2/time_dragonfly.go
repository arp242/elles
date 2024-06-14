//go:build dragonfly

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
	t := fi.Sys().(*syscall.Stat_t).Atim
	return time.Unix(t.Sec, t.Nsec)
}

// TODO: doesn't seem to have birthtime?
func Btime(absdir string, fi fs.FileInfo) time.Time {
	return time.Time{}
}
