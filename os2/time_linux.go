//go:build linux

package os2

import (
	"io/fs"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func Atime(fi fs.FileInfo) time.Time {
	if fi.Sys() == nil {
		return time.Time{}
	}
	t := fi.Sys().(*syscall.Stat_t).Atim
	return time.Unix(int64(t.Sec), int64(t.Nsec))
}

// TODO: On Linux the btime is in statx, rather than stat. We call statx() here
// because, well, it's the best we can do. But it's slow. And ugly.
//
// Should probably rewrite the stdlib bits for listing directories so we don't
// need to do this. But that's some effort and can't be bothered now, and I'm
// okay with the performance hit for now.
func Btime(absdir string, fi fs.FileInfo) time.Time {
	var s unix.Statx_t
	err := unix.Statx(0,
		filepath.Join(absdir, fi.Name()),
		unix.AT_SYMLINK_NOFOLLOW,
		unix.STATX_BTIME, &s)
	if err == nil {
		return time.Unix(s.Btime.Sec, int64(s.Btime.Nsec))
	}

	if fi.Sys() == nil {
		return time.Time{}
	}
	t := fi.Sys().(*syscall.Stat_t).Ctim
	return time.Unix(int64(t.Sec), int64(t.Nsec))
}
