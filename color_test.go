package main

import (
	"io/fs"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"zgo.at/zli"
)

func clearColors() {
	zli.WantColor = false
	for _, c := range []*string{
		&colorNormal, &colorFile, &colorDir, &colorLink, &colorPipe, &colorSocket,
		&colorBlockDev, &colorCharDev, &colorOrphan, &colorExec, &colorDoor,
		&colorSuid, &colorSgid, &colorSticky, &colorOtherWrite,
		&colorOtherWriteStick, &reset,
	} {
		*c = ""
	}
}

// Just print out stuff for manual verification; this is not likely to regress,
// and this is easier for now.
func TestDefaultColor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip() // TODO: just because of the FIFO etc.
	}

	defer clearColors()

	start(t)
	touch(t, "file")
	mkdirAll(t, "dir")
	symlink(t, "file", "link")
	mkfifo(t, "fifo")
	touch(t, "exec")
	chmod(t, 0o555, "exec")
	l, err := net.Listen("unix", "socket")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	symlink(t, "file", "link-file")
	symlink(t, "dir", "link-dir")
	symlink(t, "exec", "link-exec")
	symlink(t, "fifo", "link-fifo")
	symlink(t, "socket", "link-socket")
	symlink(t, "orphan", "link-orphan")

	touch(t, "world-file")
	touch(t, "world-dir")
	chmod(t, 0o777, "world-file")
	chmod(t, 0o777, "world-dir")

	mkdirAll(t, "sticky-dir")
	mkdirAll(t, "sticky-dir-world")
	chmod(t, 0o755|fs.ModeSticky, "sticky-dir")
	chmod(t, 0o777|fs.ModeSticky, "sticky-dir-world")

	defaultColors = "gnu"
	haveGNU := mustRun(t, "-CF", "--color=always") + "\n"
	for i, l := range strings.Split(mustRun(t, "-lF", "--color=always"), "\n") {
		if i > 0 {
			if i%5 == 0 {
				haveGNU += "\n"
			} else {
				haveGNU += " → "
			}
		}
		f := strings.Split(l, " │ ")
		haveGNU += f[2]
	}

	defaultColors = "bsd"
	haveBSD := mustRun(t, "-CF", "--color=always") + "\n"
	for i, l := range strings.Split(mustRun(t, "-lF", "--color=always"), "\n") {
		if i > 0 {
			if i%5 == 0 {
				haveBSD += "\n"
			} else {
				haveBSD += " → "
			}
		}
		f := strings.Split(l, " │ ")
		haveBSD += f[2]
	}

	// Can get the system values with the following functions, assuming it point
	// to the correct "ls".
	testGNU := func() string {
		out1, _ := exec.Command("ls", "-CF", "--color=always").CombinedOutput()
		out2, _ := exec.Command("ls", "-lF", "--color=always").CombinedOutput()
		out3 := string(out1)
		for i, l := range strings.Split(string(out2), "\n")[1:] {
			if l == "" {
				continue
			}
			if i > 0 {
				if i%5 == 0 {
					out3 += "\n"
				} else {
					out3 += " → "
				}
			}
			f := strings.Fields(l)
			if len(f) > 7 {
				out3 += strings.Join(f[8:], " ")
			}
		}
		return out3
	}
	testBSD := func() string {
		os.Setenv("CLICOLOR_FORCE", "1")
		p := "/home/martin/code/Prog/boxlike/boxlike-static"
		out1, _ := exec.Command(p, "ls", "-CFG").CombinedOutput()
		out2, _ := exec.Command(p, "ls", "-lFG").CombinedOutput()
		out3 := string(out1)
		for i, l := range strings.Split(string(out2), "\n")[1:] {
			if l == "" {
				continue
			}
			if i > 0 {
				if i%5 == 0 {
					out3 += "\n"
				} else {
					out3 += " → "
				}
			}
			f := strings.Fields(l)
			if len(f) > 7 {
				out3 += strings.Join(f[8:], " ")
			}
		}
		return out3
	}
	_, _, _, _ = testGNU, testBSD, haveGNU, haveBSD

	//fmt.Println(haveGNU)
	//fmt.Print("\n-------------------------\n\n")
	//fmt.Println(testGNU())

	//fmt.Println(haveBSD)
	//fmt.Print("\n-------------------------\n\n")
	//fmt.Println(testBSD())
}
