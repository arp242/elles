//go:build windows

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
	s := fi.Sys().(*syscall.Win32FileAttributeData).LastAccessTime
	return time.Unix(0, s.Nanoseconds())
}

func Btime(absdir string, fi fs.FileInfo) time.Time {
	if fi.Sys() == nil {
		return time.Time{}
	}
	s := fi.Sys().(*syscall.Win32FileAttributeData).CreationTime
	return time.Unix(0, s.Nanoseconds())
}
