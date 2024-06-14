//go:build !windows

package os2

import "io/fs"

func Hidden(absdir string, fi fs.DirEntry) bool {
	return fi.Name()[0] == '.'
}
