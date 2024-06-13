//go:build unix && !freebsd

package os2

import "golang.org/x/sys/unix"

func Mkfifo(path string, mode uint32) error         { return unix.Mkfifo(path, mode) }
func Mknod(path string, mode uint32, dev int) error { return unix.Mknod(path, mode, dev) }
