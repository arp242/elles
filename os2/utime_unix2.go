//go:build unix && (386 || arm)

package os2

import (
	"time"

	"golang.org/x/sys/unix"
)

func Utimes(path string, atime, mtime time.Time) error {
	ts := make([]unix.Timespec, 2)
	if !atime.IsZero() {
		ts[0] = unix.Timespec{Sec: int32(atime.Unix()), Nsec: int32(atime.Nanosecond())}
	}
	if !mtime.IsZero() {
		ts[1] = unix.Timespec{Sec: int32(mtime.Unix()), Nsec: int32(mtime.Nanosecond())}
	}
	return unix.UtimesNanoAt(unix.AT_FDCWD, path, ts, unix.AT_SYMLINK_NOFOLLOW)
}
