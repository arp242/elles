//go:build solaris

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

// TODO: we need to use getattrat()/fgetattr() to get this, with A_CRTIME. But
// this isn't exposed in syscall or x/sys/unix.
func Btime(absdir string, fi fs.FileInfo) time.Time {
	return time.Time{}
}
