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
	zli.WantColor, colorLinkAsTarget = false, false
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

	os.Setenv("ELLES_COLORS", "gnu")
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

	os.Setenv("ELLES_COLORS", "bsd")
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

	os.Unsetenv("LS_COLORS")

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

// t.Run("color-dtype-dir", func(t *testing.T) {
// 	// Ensure "ls --color" properly colors other-writable and sticky directories.
// 	// Before coreutils-6.2, this test would fail, coloring all three
// 	// directories the same as the first one -- but only on a file system
// 	// with dirent.d_type support.

// 	start(t)

// 	// mkdir d other-writable sticky
// 	// chmod o+w other-writable
// 	// chmod o+t sticky
// 	//
// 	//
// 	// TERM=xterm ls --color=always > out
// 	// cat -A out > o1
// 	// mv o1 out
// 	//
// 	// cat <<\EOF > exp
// 	// ^[[0m^[[01;34md^[[0m$
// 	// ^[[34;42mother-writable^[[0m$
// 	// out$
// 	// ^[[37;44msticky^[[0m$
// 	// EOF
// 	//
// 	// compare exp out
// 	//
// 	// rm exp
// 	//
// 	// # Turn off colors for other-writable dirs and ensure
// 	// # we fall back to the color for standard directories.
// 	//
// 	// LS_COLORS="ow=:" ls --color=always > out
// 	// cat -A out > o1
// 	// mv o1 out
// 	//
// 	// cat <<\EOF > exp
// 	// ^[[0m^[[01;34md^[[0m$
// 	// ^[[01;34mother-writable^[[0m$
// 	// out$
// 	// ^[[37;44msticky^[[0m$
// 	// EOF
// 	//
// 	// compare exp out
// })

// t.Run("color-norm", func(t *testing.T) {
// 	// Ensure "ls --color" properly colors "normal" text and files. I.e.,
// 	// that it uses NORMAL to style non file name output and file names with
// 	// no associated color (unless FILE is also set).

// 	start(t)

// 	// # Output time as something constant
// 	// export TIME_STYLE="+norm"
// 	//
// 	// # helper to strip ls columns up to "norm" time
// 	// qls() { sed 's/-r.*norm/norm/'; }
// 	//
// 	// touch exe
// 	// chmod u+x exe
// 	// touch nocolor
// 	//
// 	// TCOLORS="no=7:ex=01;32"
// 	//
// 	// # Uncolored file names inherit NORMAL attributes.
// 	// LS_COLORS=$TCOLORS      ls -gGU --color exe nocolor | qls >> out
// 	// LS_COLORS=$TCOLORS      ls -xU  --color exe nocolor       >> out
// 	// LS_COLORS=$TCOLORS      ls -gGU --color nocolor exe | qls >> out
// 	// LS_COLORS=$TCOLORS      ls -xU  --color nocolor exe       >> out
// 	//
// 	// # NORMAL does not override FILE though
// 	// LS_COLORS=$TCOLORS:fi=1 ls -gGU --color nocolor exe | qls >> out
// 	//
// 	// # Support uncolored ordinary files that do _not_ inherit from NORMAL.
// 	// # Note there is a redundant RESET output before a non colored
// 	// # file in this case which may be removed in future.
// 	// LS_COLORS=$TCOLORS:fi=  ls -gGU --color nocolor exe | qls >> out
// 	// LS_COLORS=$TCOLORS:fi=0 ls -gGU --color nocolor exe | qls >> out
// 	//
// 	// # A caveat worth noting is that commas (-m), indicator chars (-F)
// 	// # and the "total" line, do not currently use NORMAL attributes
// 	// LS_COLORS=$TCOLORS      ls -mFU --color nocolor exe       >> out
// 	//
// 	// # Ensure no coloring is done unless enabled
// 	// LS_COLORS=$TCOLORS      ls -gGU         nocolor exe | qls >> out
// 	//
// 	// cat -A out > out.display
// 	// mv out.display out
// 	//
// 	// cat <<\EOF > exp
// 	// ^[[0m^[[7mnorm ^[[m^[[01;32mexe^[[0m$
// 	// ^[[7mnorm nocolor^[[0m$
// 	// ^[[0m^[[7m^[[m^[[01;32mexe^[[0m  ^[[7mnocolor^[[0m$
// 	// ^[[0m^[[7mnorm nocolor^[[0m$
// 	// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
// 	// ^[[0m^[[7mnocolor^[[0m  ^[[7m^[[m^[[01;32mexe^[[0m$
// 	// ^[[0m^[[7mnorm ^[[m^[[1mnocolor^[[0m$
// 	// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
// 	// ^[[0m^[[7mnorm ^[[m^[[mnocolor^[[0m$
// 	// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
// 	// ^[[0m^[[7mnorm ^[[m^[[0mnocolor^[[0m$
// 	// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
// 	// ^[[0m^[[7mnocolor^[[0m, ^[[7m^[[m^[[01;32mexe^[[0m*$
// 	// norm nocolor$
// 	// norm exe$
// 	// EOF
// 	//
// 	// compare exp out
// })

// t.Run("multihardlink", func(t *testing.T) {
// 	// Ensure "ls --color" properly colors names of hard linked files.
// 	start(t)

// 	// touch file file1
// 	// ln file1 file2 || skip_ "can't create hard link"
// 	// code_mh='44;37'
// 	// code_ex='01;32'
// 	// code_png='01;35'
// 	// c0=$(printf '\033[0m')
// 	// c_mh=$(printf '\033[%sm' $code_mh)
// 	// c_ex=$(printf '\033[%sm' $code_ex)
// 	// c_png=$(printf '\033[%sm' $code_png)

// 	// # regular file - not hard linked
// 	// LS_COLORS="mh=$code_mh" ls -U1 --color=always file > out
// 	// printf "file\n" > out_ok
// 	// compare out out_ok

// 	// # hard links
// 	// LS_COLORS="mh=$code_mh" ls -U1 --color=always file1 file2 > out
// 	// printf "$c0${c_mh}file1$c0
// 	// ${c_mh}file2$c0
// 	// " > out_ok
// 	// compare out out_ok

// 	// # hard links and png (hard link coloring takes precedence)
// 	// mv file2 file2.png
// 	// LS_COLORS="mh=$code_mh:*.png=$code_png" ls -U1 --color=always file1 file2.png \
// 	//   > out
// 	// printf "$c0${c_mh}file1$c0
// 	// ${c_mh}file2.png$c0
// 	// " > out_ok
// 	// compare out out_ok

// 	// # hard links and exe (exe coloring takes precedence)
// 	// chmod a+x file2.png
// 	// LS_COLORS="mh=$code_mh:*.png=$code_png:ex=$code_ex" \
// 	//   ls -U1 --color=always file1 file2.png > out
// 	// chmod a-x file2.png
// 	// printf "$c0${c_ex}file1$c0
// 	// ${c_ex}file2.png$c0
// 	// " > out_ok
// 	// compare out out_ok

// 	// # hard links and png (hard link coloring disabled => png coloring enabled)
// 	// LS_COLORS="mh=00:*.png=$code_png" ls -U1 --color=always file1 file2.png > out \

// 	// printf "file1
// 	// $c0${c_png}file2.png$c0
// 	// " > out_ok
// 	// compare out out_ok

// 	// # hard links and png (hard link coloring not enabled explicitly => png coloring)
// 	// LS_COLORS="*.png=$code_png" ls -U1 --color=always file1 file2.png > out \

// 	// printf "file1
// 	// $c0${c_png}file2.png$c0
// 	// " > out_ok
// 	// compare out out_ok
// })
