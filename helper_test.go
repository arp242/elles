package main

import (
	"bytes"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"zgo.at/elles/os2"
	"zgo.at/zli"
)

func init() {
	zli.WantColor = false
	os.Unsetenv("LS_COLORS")
	os.Unsetenv("LSCOLORS")
	os.Setenv("COLUMNS", "80")
	columns = 80
}

var mydir = func() string {
	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return d
}()

type uinfo struct {
	Username, Groupname string
	UserID, GroupID     string
	UID, GID            int // Unix only
}

var userinfo = func() uinfo {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		panic(err)
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	return uinfo{
		Username:  u.Username,
		Groupname: g.Name,
		UserID:    u.Uid,
		GroupID:   g.Gid,
		UID:       uid,
		GID:       gid,
	}
}()

// Get size of (empty) direcrory; this differs per filesystem.
//
// To test with e.g. XFS on Linux:
//
// dd if=/dev/zero of=/tmp/img bs=1M count=300
// mkfs.xfs /tmp/img ./tmp
// doas mount /tmp/img ./tmp
// doas chown martin tmp
//
// export TMPDIR=./tmp/tmp
// go test
var dirsize = func() int {
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	st, err := os.Stat(tmp)
	if err != nil {
		panic(err)
	}
	return int(st.Size())
}()

var join = filepath.Join

func run(t *testing.T, args ...string) (o string, ok bool) {
	zli.Test(t)
	defer func() {
		ok = recover() == nil
		o = strings.TrimSuffix(zli.Stdout.(*bytes.Buffer).String(), "\n")
	}()
	os.Args = append([]string{"elles"}, args...)
	main()
	return o, ok
}

func mustRun(t *testing.T, args ...string) string {
	t.Helper()
	out, ok := run(t, args...)
	if !ok {
		t.Fatalf("mustRun failed: %v", out)
	}
	return out
}

// Replacement patterns:
//
// martin   - username
// tournoij - group name

func norm(s string, repl ...string) string {
	if len(repl)%2 == 1 {
		panic("norm: uneven repl")
	}
	s = strings.TrimPrefix(s, "\n")
	s = strings.ReplaceAll(s, "\t", "")

	repl = append(repl,
		"martin", userinfo.Username,
		"tournoij", userinfo.Groupname,
	)
	s = strings.NewReplacer(repl...).Replace(s)
	return s
}

// Start a test by creating a new temporary directory and cd'ing to it. Register
// a cleanup function to cd back to the previous directory: this is mostly
// needed for Windows and illumos, who will refuse to delete the directory
// otherwise causing the test to fail.
func start(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	cd(t, tmp)
	t.Cleanup(func() { cd(t, mydir) })
	return tmp
}

func isCI() bool {
	_, ok := os.LookupEnv("CI")
	return ok
}

func supportsSparseFiles(t *testing.T, skip bool) bool {
	tmp := t.TempDir()

	createSparse(t, 8192, tmp, ".sparse-test")
	st, err := os.Stat(join(tmp, ".sparse-test"))
	if err != nil {
		t.Fatal(err)
	}
	if os2.Blocks(st) != 0 {
		if skip {
			t.Skip("filesystem doesn't appear to support sparse files")
		}
		return false
	}
	return true
}

func supportsFIFO(t *testing.T, skip bool) bool {
	if runtime.GOOS == "windows" {
		if skip {
			t.Skipf("%s does not support named sockets (FIFO)", runtime.GOOS)
		}
		return false
	}
	return true
}

func supportsDevice(t *testing.T, skip bool) bool {
	switch runtime.GOOS {
	case "windows":
		if skip {
			t.Skipf("%s does not support device nodes", runtime.GOOS)
		}
		return false
	case "freebsd", "netbsd", "openbsd", "dragonfly", "darwin", "illumos", "solaris":
		if skip {
			t.Skipf("%s requires root permissions to create device nodes", runtime.GOOS)
		}
		return false
	}
	return true
}

func supportsBtime(t *testing.T, skip bool) bool {
	switch runtime.GOOS {
	case "openbsd", "dragonfly", "illumos", "solaris":
		if skip {
			t.Skipf("btime not supported on %s", runtime.GOOS)
		}
		return false
	}
	return true
}

func supportsUtimes(t *testing.T, skip bool) bool {
	if runtime.GOOS == "windows" {
		if skip {
			t.Skipf("%s does not support os2.Utime", runtime.GOOS)
		}
		return false
	}
	return true
}

// cd
func cd(t testing.TB, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("cd: path must have at least one element: %s", path)
	}
	err := os.Chdir(join(path...))
	if err != nil {
		t.Fatalf("cd(%q): %s", join(path...), err)
	}
}

// pwd
func pwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("pwd: %s", err)
	}
	return filepath.ToSlash(wd)
}

func createSparse(t *testing.T, sz int64, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("createSparse: path must have at least one element: %s", path)
	}
	fp, err := os.Create(join(path...))
	if err != nil {
		t.Fatalf("createSparse(%q): %s", join(path...), err)
	}
	if err := fp.Truncate(sz); err != nil {
		t.Fatalf("createSparse(%q): %s", join(path...), err)
	}
	if err := fp.Close(); err != nil {
		t.Fatalf("createSparse(%q): %s", join(path...), err)
	}
}

// ls
func list(t *testing.T, path ...string) []string {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("mkdirAll: path must have at least one element: %s", path)
	}
	ls, err := os.ReadDir(join(path...))
	if err != nil {
		t.Fatalf("list(%q): %s", join(path...), err)
	}
	list := make([]string, 0, len(ls))
	for _, f := range ls {
		if n := f.Name(); n[0] != '.' {
			list = append(list, mustRun(t, "-1d", n))
		}
	}
	return list
}

// mkdir -p
func mkdirAll(t *testing.T, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("mkdirAll: path must have at least one element: %s", path)
	}
	err := os.MkdirAll(join(path...), 0o0755)
	if err != nil {
		t.Fatalf("mkdirAll(%q): %s", join(path...), err)
	}
}

// touch
func touch(t testing.TB, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("touch: path must have at least one element: %s", path)
	}
	fp, err := os.Create(join(path...))
	if err != nil {
		t.Fatalf("touch(%q): %s", join(path...), err)
	}
	err = fp.Close()
	if err != nil {
		t.Fatalf("touch(%q): %s", join(path...), err)
	}
}

// touch -d
func touchDate(t testing.TB, tt time.Time, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("touch: path must have at least one element: %s", path)
	}
	fp, err := os.Create(join(path...))
	if err != nil {
		t.Fatalf("touch(%q): %s", join(path...), err)
	}
	err = fp.Close()
	if err != nil {
		t.Fatalf("touch(%q): %s", join(path...), err)
	}
	err = os2.Utimes(join(path...), tt, tt)
	if err != nil {
		t.Fatalf("touch(%q): %s", join(path...), err)
	}
}

// ln -s
func symlink(t *testing.T, target string, link ...string) {
	t.Helper()
	if len(link) < 1 {
		t.Fatalf("symlink: link must have at least one element: %s", link)
	}
	err := os.Symlink(target, join(link...))
	if err != nil {
		t.Fatalf("symlink(%q, %q): %s", target, join(link...), err)
	}
}

// mkfifo
func mkfifo(t *testing.T, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("mkfifo: path must have at least one element: %s", path)
	}
	err := os2.Mkfifo(join(path...), 0o644)
	if err != nil {
		t.Fatalf("mkfifo(%q): %s", join(path...), err)
	}
}

// mknod
func mknod(t *testing.T, dev int, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("mknod: path must have at least one element: %s", path)
	}
	err := os2.Mknod(join(path...), 0o644, dev)
	if err != nil {
		t.Fatalf("mknod(%d, %q): %s", dev, join(path...), err)
	}
}

// chmod
func chmod(t *testing.T, mode fs.FileMode, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("chmod: path must have at least one element: %s", path)
	}
	err := os.Chmod(join(path...), mode)
	if err != nil {
		t.Fatalf("chmod(%q): %s", join(path...), err)
	}
}

// rm
func rm(t *testing.T, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("rm: path must have at least one element: %s", path)
	}
	err := os.Remove(join(path...))
	if err != nil {
		t.Fatalf("rm(%q): %s", join(path...), err)
	}
}

// rm -r
func rmAll(t *testing.T, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("rmAll: path must have at least one element: %s", path)
	}
	err := os.RemoveAll(join(path...))
	if err != nil {
		t.Fatalf("rmAll(%q): %s", join(path...), err)
	}
}

// echo > and echo >>
func echoAppend(t *testing.T, data string, path ...string) { t.Helper(); echo(t, false, data, path...) }
func echoTrunc(t *testing.T, data string, path ...string)  { t.Helper(); echo(t, true, data, path...) }
func echo(t *testing.T, trunc bool, data string, path ...string) {
	n := "echoAppend"
	if trunc {
		n = "echoTrunc"
	}
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("%s: path must have at least one element: %s", n, path)
	}

	err := func() error {
		var (
			fp  *os.File
			err error
		)
		if trunc {
			fp, err = os.Create(join(path...))
		} else {
			fp, err = os.OpenFile(join(path...), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		}
		if err != nil {
			return err
		}
		if err := fp.Sync(); err != nil {
			return err
		}
		if _, err := fp.WriteString(data); err != nil {
			return err
		}
		if err := fp.Sync(); err != nil {
			return err
		}
		return fp.Close()
	}()
	if err != nil {
		t.Fatalf("%s(%q): %s", n, join(path...), err)
	}
}
