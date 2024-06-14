//go:build windows

package os2

import (
	"io/fs"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func Hidden(absdir string, fi fs.DirEntry) bool {
	if fi.Name()[0] == '.' {
		return true
	}

	p := filepath.Join(absdir, fi.Name())
	ptr, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return false
	}
	attr, err := windows.GetFileAttributes(ptr)
	if err != nil {
		// TODO: fails on e.g. C:\pagefile.sys with:
		// The process cannot access the file because it is being used by another process.
		//
		// dir C:\ doesn't display it, so there must be some way to get this(?)
		//zli.Errorf("windows.GetFileAttributes: %q: %s", p, err)
		return false
	}
	return attr&windows.FILE_ATTRIBUTE_HIDDEN != 0
}
