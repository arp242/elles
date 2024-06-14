//go:build unix

package os2

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"zgo.at/zli"
)

// TODO: unused at the moment; ls -l prints this, but I don't think I've ever
// used it in 25 years. Should maybe add option though.
func Numlinks(absdir string, fi fs.FileInfo) uint64 {
	if fi.Sys() == nil {
		return 0
	}
	return uint64(fi.Sys().(*syscall.Stat_t).Nlink)
}

func OwnerID(absdir string, fi fs.FileInfo) (string, string) {
	if fi.Sys() == nil {
		return "", ""
	}
	s := fi.Sys().(*syscall.Stat_t)
	return strconv.FormatUint(uint64(s.Uid), 10), strconv.FormatUint(uint64(s.Gid), 10)
}

func Serial(absdir string, fi fs.FileInfo) uint64 {
	if fi.Sys() == nil {
		return 0
	}
	return fi.Sys().(*syscall.Stat_t).Ino
}

func Blocks(fi fs.FileInfo) int64 {
	if fi.Sys() == nil {
		return -1
	}
	return fi.Sys().(*syscall.Stat_t).Blocks
}

func IsELOOP(err error) bool {
	var pErr *fs.PathError
	if errors.As(err, &pErr) {
		return errors.Is(pErr.Err, syscall.ELOOP)
	}
	return false
}

var (
	mnts     []string
	blocks   []int
	mntsOnce sync.Once
)

// Linux and illumos.
func mntsProc() bool {
	var fp *os.File
	for _, f := range []string{
		"/proc/mounts", // Linux
		"/etc/mnttab",  // illumos
	} {
		var err error
		fp, err = os.Open(f)
		if err == nil {
			break
		}
	}
	if fp == nil {
		return false
	}
	defer fp.Close()

	scan := bufio.NewScanner(fp)
	mnts = make([]string, 0, 4)
	for scan.Scan() {
		l := strings.Fields(scan.Text())
		if l[0] != "cgroup" && l[0] != "cgroup2" {
			mnts = append(mnts, l[1])
		}
	}
	return true
}

// Other Unix. TODO: there's probably a better way.
func mntsCmd() bool {
	out, err := exec.Command("mount").CombinedOutput()
	if err != nil {
		return false
	}
	for _, l := range strings.Split(string(out), "\n") {
		f := strings.Fields(l)
		if len(f) >= 3 && f[0] != "cgroup" && f[0] != "cgroup2" && f[0] != "map" {
			mnts = append(mnts, f[2])
		}
	}
	return true
}

// Get the block size for the filesystem on which path resides.
func Blocksize(path string) int {
	mntsOnce.Do(func() {
		if !mntsProc() && !mntsCmd() {
			return
		}

		blocks = make([]int, 0, len(mnts))
		for _, m := range mnts {
			bsize, err := statfs(m)
			if err != nil {
				zli.Errorf("statfs %q: %s", m, err)
				bsize = 512
			}

			// statvfs also has f_frsize, for "Fundamental file system block
			// size", but I'm not really sure when/why to use this over f_bsize.
			// FreeBSD manpage has "minimum unit of allocation on this file
			// system. (This corresponds to the f_bsize member of struct
			// statfs)", so it's always the same? POSIX doesn't say much either.
			// So idk.
			blocks = append(blocks, bsize)
		}
	})

	for i, m := range mnts {
		if strings.HasPrefix(path, m) {
			return blocks[i]
		}
	}

	// This should never happen, but print warning just in case.
	zli.Errorf("blocksize: no blocksize found for %q; defaulting to 512", path)
	return 512
}
