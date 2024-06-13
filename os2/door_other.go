//go:build !solaris

package os2

import "io/fs"

func IsDoor(fi fs.FileInfo) bool { return false }
