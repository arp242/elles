//go:build !unix

package os2

import "time"

func Utimes(path string, atime, mtime time.Time) error {
	// SetFileInformationByHandle()?
	panic("not implemented")
}
