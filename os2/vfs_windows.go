//go:build windows

package os2

import (
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"zgo.at/zli"
)

func Numlinks(absdir string, fi fs.FileInfo) int {
	fp, err := os.Open(filepath.Join(absdir, fi.Name()))
	if err != nil {
		return 1
	}
	defer fp.Close()

	var info windows.ByHandleFileInformation
	err = windows.GetFileInformationByHandle(windows.Handle(fp.Fd()), &info)
	if err != nil {
		zli.Errorf(err)
		return 1
	}
	return int(info.NumberOfLinks)
}

func OwnerID(absdir string, fi fs.FileInfo) (string, string) {
	fp, err := os.Open(filepath.Join(absdir, fi.Name()))
	if err != nil {
		return "", ""
	}
	defer fp.Close()

	sec, err := windows.GetSecurityInfo(windows.Handle(fp.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.GROUP_SECURITY_INFORMATION)
	if err != nil {
		return "", ""
	}

	var owner, group string
	if o, _, err := sec.Owner(); err == nil {
		owner = o.String()
	}
	if g, _, err := sec.Group(); err == nil {
		group = g.String()
	}
	return owner, group
}

func Serial(absdir string, fi fs.FileInfo) uint64 {
	fp, err := os.Open(filepath.Join(absdir, fi.Name()))
	if err != nil {
		return 0
	}
	defer fp.Close()

	var info windows.ByHandleFileInformation
	err = windows.GetFileInformationByHandle(windows.Handle(fp.Fd()), &info)
	if err != nil {
		zli.Errorf(err)
		return 0
	}
	return (uint64(info.FileIndexHigh) << 32) | uint64(info.FileIndexLow)
}

func Blocks(fi fs.FileInfo) int64 {
	// TODO: not sure how to get this.
	return fi.Size() / 512
}

func Blocksize(path string) int {
	// TODO: not sure how to get this.
	return 512
}

func IsELOOP(err error) bool {
	// TODO: can this happen on Windows?
	return false
}
