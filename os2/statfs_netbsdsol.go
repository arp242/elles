//go:build netbsd || solaris

package os2

import "golang.org/x/sys/unix"

func statfs(m string) (int, error) {
	var vfs unix.Statvfs_t
	err := unix.Statvfs(m, &vfs)
	return int(vfs.Bsize), err
}
