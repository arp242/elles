//go:build !unix

package os2

import (
	"fmt"
	"runtime"
)

func Mkfifo(path string, mode uint32) error {
	return fmt.Errorf("no FIFOs on %s", runtime.GOOS)
}
func Mknod(path string, mode uint32, dev int) error {
	return fmt.Errorf("no device nodes on %s", runtime.GOOS)
}
