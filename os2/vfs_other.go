//go:build !unix && !windows

package os2

import "io/fs"

// No-ops for platforms we don't really support. Most of this isn't really
// critical, so okay to return dummy values.

func Numlinks(absdir string, fi fs.FileInfo) int             { return 1 }
func OwnerID(absdir string, fi fs.FileInfo) (string, string) { return "", "" }
func Serial(absdir string, fi fs.FileInfo) uint64            { return 0 }
func Blocksize(path string) int                              { return 512 }
func Blocks(fi fs.FileInfo) int64                            { return fi.Size() / 512 }
func IsELOOP(err error) bool                                 { return false }
