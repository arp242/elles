//go:build solaris

package os2

import (
	"io/fs"

	"golang.org/x/sys/unix"
)

func IsDoor(fi fs.FileInfo) bool { return fi.Mode()&unix.S_IFDOOR != 0 }
