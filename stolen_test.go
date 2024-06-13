package main

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// These are tests from FreeBSD (commit 0dfd11abc) and GNU coreutils (commit
// bbc972b), and should be merged/converted in to main_test.go

func TestFreeBSD(t *testing.T) {
	createTestInputs := func(t *testing.T) string {
		if runtime.GOOS == "windows" {
			// TODO: rewrite these tests to not depend on this one function.
			t.Skip("control characters aren't permitted in Windows")
		}

		tmp := start(t)

		mkdirAll(t, "a/b/1")   // mkdir -m 0755 -p a/b/1
		symlink(t, "a/b", "c") // ln -s a/b c
		touch(t, "d")
		symlink(t, "d", "e") // ln d e
		touch(t, ".f")
		mkdirAll(t, ".g")
		//mkfifo(t,  "h")
		if err := os.WriteFile("i", []byte(strings.Repeat("\x00", 1000)), 0o644); err != nil {
			// dd if=/dev/zero of=i count=1000 bs=1
			t.Fatal(err)
		}
		touch(t, "klmn")
		touch(t, "opqr")
		touch(t, "stuv")
		touch(t, "wxyz") // install -m 0755 /dev/null wxyz
		chmod(t, 0o755, "wxyz")

		for _, n := range []byte{0b00000001, 0b00000010, 0b00000011, 0b00000100,
			0b00000101, 0b00000110, 0b00000111, 0b00001000, 0b00001001, 0b00001010,
			0b00001011, 0b00001100, 0b00001101, 0b00001110, 0b00001111} {
			touch(t, string([]byte{n}))
		}

		return tmp
	}

	t.Run("ls -B prints out non-printable characters", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows doesn't allow non-printable characters")
		}
		start(t)

		touch(t, "y\013z")
		have := mustRun(t, "-1")
		want := `y$'\x0b'z`
		if have != want {
			t.Errorf("\nhave: %q\nwant: %q", have, want)
		}
	})

	t.Run("ls -C is multi-column, sorted down", func(t *testing.T) {
		createTestInputs(t)
		restore := columns
		defer func() { columns = restore }()
		columns = 40

		have := mustRun(t, "-C")
		want := strings.ReplaceAll(`
			$'\x01'  $'\x06'  $'\x0b'  a  klmn
			$'\x02'  $'\x07'  $'\x0c'  c  opqr
			$'\x03'  $'\x08'  $'\r'    d  stuv
			$'\x04'  $'\t'    $'\x0e'  e  wxyz
			$'\x05'  $'\n'    $'\x0f'  i`[1:], "\t", "")
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
		}
	})

	t.Run("ls -R prints out the directory contents recursively", func(t *testing.T) {
		createTestInputs(t)

		have := mustRun(t, "-1R")
		want := strings.ReplaceAll(`
			.:
			$'\x01'
			$'\x02'
			$'\x03'
			$'\x04'
			$'\x05'
			$'\x06'
			$'\x07'
			$'\x08'
			$'\t'
			$'\n'
			$'\x0b'
			$'\x0c'
			$'\r'
			$'\x0e'
			$'\x0f'
			a
			c
			d
			e
			i
			klmn
			opqr
			stuv
			wxyz

			a:
			b

			a/b:
			1

			a/b/1:`[1:], "\t", "")
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
		}
	})

	t.Run("-d doesn't descend down directories", func(t *testing.T) {
		start(t)
		mkdirAll(t, "a/b")

		for _, p := range []string{".", pwd(t), "a"} {
			o := mustRun(t, "-1d", p)
			if o != p {
				t.Fatalf("output not equal:\npath: %q\nout:  %q", p, o)
			}
		}
	})

	t.Run("ls -t sorts by modification time", func(t *testing.T) {
		start(t)
		touch(t, "a")
		time.Sleep(10 * time.Millisecond) // If we don't sleep it both are written at the same time.
		touch(t, "b")

		{
			have := strings.Split(mustRun(t, "-1t"), "\n")
			want := []string{"b", "a"}
			if !reflect.DeepEqual(have, want) {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}

		rm(t, "a")
		touch(t, "a")

		{
			have := strings.Split(mustRun(t, "-1t"), "\n")
			want := []string{"a", "b"}
			if !reflect.DeepEqual(have, want) {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}
	})

	t.Run("ls -u sorts by last access", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			t.Skip("TODO: dunno why this fails; atime seems weird on macOS")
		}
		if runtime.GOOS == "windows" {
			t.Skip("TODO: fix")
		}

		start(t)
		touch(t, "a")
		time.Sleep(10 * time.Millisecond) // If we don't sleep it both are written at the same time.
		touch(t, "b")

		{
			have := strings.Split(mustRun(t, "-1tu"), "\n")
			want := []string{"b", "a"}
			if !reflect.DeepEqual(have, want) {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}

		if _, err := os.ReadFile("a"); err != nil { // cat a.file
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
		echoAppend(t, "i am a", "b")

		{
			have := strings.Split(mustRun(t, "-1tu"), "\n")
			want := []string{"a", "b"}
			if !reflect.DeepEqual(have, want) {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}
	})

	t.Run("ls -v sorts based on strverscmp(3)", func(t *testing.T) {
		start(t)
		for _, f := range []string{"000", "00", "01", "010", "09", "0", "1", "9", "10"} {
			touch(t, f)
		}

		have := strings.Split(mustRun(t, "-1v"), "\n")
		//want := []string{"000", "00", "01", "010", "09", "0", "1", "9", "10"}
		want := []string{"000", "00", "0", "01", "09", "010", "1", "9", "10"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %q\nwant: %q", have, want)
		}
	})
}

func TestGNU(t *testing.T) {
	t.Run("color-dtype-dir", func(t *testing.T) {
		// Ensure "ls --color" properly colors other-writable and sticky directories.
		// Before coreutils-6.2, this test would fail, coloring all three
		// directories the same as the first one -- but only on a file system
		// with dirent.d_type support.

		start(t)

		// mkdir d other-writable sticky
		// chmod o+w other-writable
		// chmod o+t sticky
		//
		//
		// TERM=xterm ls --color=always > out
		// cat -A out > o1
		// mv o1 out
		//
		// cat <<\EOF > exp
		// ^[[0m^[[01;34md^[[0m$
		// ^[[34;42mother-writable^[[0m$
		// out$
		// ^[[37;44msticky^[[0m$
		// EOF
		//
		// compare exp out
		//
		// rm exp
		//
		// # Turn off colors for other-writable dirs and ensure
		// # we fall back to the color for standard directories.
		//
		// LS_COLORS="ow=:" ls --color=always > out
		// cat -A out > o1
		// mv o1 out
		//
		// cat <<\EOF > exp
		// ^[[0m^[[01;34md^[[0m$
		// ^[[01;34mother-writable^[[0m$
		// out$
		// ^[[37;44msticky^[[0m$
		// EOF
		//
		// compare exp out
	})

	t.Run("color-norm", func(t *testing.T) {
		// Ensure "ls --color" properly colors "normal" text and files. I.e.,
		// that it uses NORMAL to style non file name output and file names with
		// no associated color (unless FILE is also set).

		start(t)

		// # Output time as something constant
		// export TIME_STYLE="+norm"
		//
		// # helper to strip ls columns up to "norm" time
		// qls() { sed 's/-r.*norm/norm/'; }
		//
		// touch exe
		// chmod u+x exe
		// touch nocolor
		//
		// TCOLORS="no=7:ex=01;32"
		//
		// # Uncolored file names inherit NORMAL attributes.
		// LS_COLORS=$TCOLORS      ls -gGU --color exe nocolor | qls >> out
		// LS_COLORS=$TCOLORS      ls -xU  --color exe nocolor       >> out
		// LS_COLORS=$TCOLORS      ls -gGU --color nocolor exe | qls >> out
		// LS_COLORS=$TCOLORS      ls -xU  --color nocolor exe       >> out
		//
		// # NORMAL does not override FILE though
		// LS_COLORS=$TCOLORS:fi=1 ls -gGU --color nocolor exe | qls >> out
		//
		// # Support uncolored ordinary files that do _not_ inherit from NORMAL.
		// # Note there is a redundant RESET output before a non colored
		// # file in this case which may be removed in future.
		// LS_COLORS=$TCOLORS:fi=  ls -gGU --color nocolor exe | qls >> out
		// LS_COLORS=$TCOLORS:fi=0 ls -gGU --color nocolor exe | qls >> out
		//
		// # A caveat worth noting is that commas (-m), indicator chars (-F)
		// # and the "total" line, do not currently use NORMAL attributes
		// LS_COLORS=$TCOLORS      ls -mFU --color nocolor exe       >> out
		//
		// # Ensure no coloring is done unless enabled
		// LS_COLORS=$TCOLORS      ls -gGU         nocolor exe | qls >> out
		//
		// cat -A out > out.display
		// mv out.display out
		//
		// cat <<\EOF > exp
		// ^[[0m^[[7mnorm ^[[m^[[01;32mexe^[[0m$
		// ^[[7mnorm nocolor^[[0m$
		// ^[[0m^[[7m^[[m^[[01;32mexe^[[0m  ^[[7mnocolor^[[0m$
		// ^[[0m^[[7mnorm nocolor^[[0m$
		// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
		// ^[[0m^[[7mnocolor^[[0m  ^[[7m^[[m^[[01;32mexe^[[0m$
		// ^[[0m^[[7mnorm ^[[m^[[1mnocolor^[[0m$
		// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
		// ^[[0m^[[7mnorm ^[[m^[[mnocolor^[[0m$
		// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
		// ^[[0m^[[7mnorm ^[[m^[[0mnocolor^[[0m$
		// ^[[7mnorm ^[[m^[[01;32mexe^[[0m$
		// ^[[0m^[[7mnocolor^[[0m, ^[[7m^[[m^[[01;32mexe^[[0m*$
		// norm nocolor$
		// norm exe$
		// EOF
		//
		// compare exp out
	})

	t.Run("group-dirs", func(t *testing.T) { // --group-directories-first
		start(t)

		mkdirAll(t, "dir/b")
		touch(t, "dir/a")
		symlink(t, "b", "dir/bl")

		have := strings.Fields(mustRun(t, "--group-dirs", "dir"))
		want := []string{"b", "bl", "a"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}

		have = strings.Fields(mustRun(t, "--group-dir", "-d", "dir/a", "dir/b", "dir/bl"))
		want = []string{"dir/b", "dir/bl", "dir/a"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	})

	t.Run("hyperlink", func(t *testing.T) { // Test --hyperlink processing
		start(t)

		// # lookup based on first letter
		// encode() {
		//  printf '%s\n' \
		//   'sp%20ace' 'ques%3ftion' 'back%5cslash' 'encoded%253Fquestion' 'testdir' \
		//   "$1" |
		//  sort -k1,1.1 -s | uniq -w1 -d
		// }
		//
		// ls_encoded() {
		//   ef=$(encode "$1")
		//   echo "$ef" | grep 'dir$' >/dev/null && dir=: || dir=''
		//   printf '\033]8;;file:///%s\a%s\033]8;;\a%s\n' \
		//     "$ef" "$1" "$dir"
		// }
		//
		// # These could be encoded, so remove from consideration
		// strip_host_and_path() {
		//   sed 's|file://.*/|file:///|'
		// }
		//
		// mkdir testdir
		// (
		// cd testdir
		// ls_encoded "testdir" > ../exp.t
		// for f in 'back\slash' 'encoded%3Fquestion' 'ques?tion' 'sp ace'; do
		//   touch "$f"
		//   ls_encoded "$f" >> ../exp.t
		// done
		// )
		// ln -s testdir testdirl
		// (cat exp.t && printf '\n' && sed 's/[^\/]testdir/&l/' exp.t) > exp \
		//
		// ls --hyper testdir testdirl >out.t
		// strip_host_and_path <out.t >out
		// compare exp out
		//
		// ln -s '/probably_missing' testlink
		// ls -l --hyper testlink > out.t
		// strip_host_and_path <out.t >out
		// grep 'file:///probably_missing' out
	})

	t.Run("ls-time", func(t *testing.T) { // Test some of ls's sorting options.
		start(t)

		// # Avoid any possible glitches due to daylight-saving changes near the
		// # timestamps used during the test.
		// TZ=UTC0
		// export TZ
		//
		// t1='1998-01-15 21:00'
		// t2='1998-01-15 22:00'
		// t3='1998-01-15 23:00'
		//
		// u1='1998-01-14 11:00'
		// u2='1998-01-14 12:00'
		// u3='1998-01-14 13:00'
		//
		// touch -m -d "$t3" a
		// touch -m -d "$t2" b
		// touch -m -d "$t1" c
		//
		// touch -a -d "$u3" c
		// touch -a -d "$u2" b
		// # Make sure A has ctime at least 1 second more recent than C's.
		// sleep 2
		// touch -a -d "$u1" a
		// # Updating the atime is usually enough to update the ctime, but on
		// # Solaris 10's tmpfs, ctime is not updated, so force an update here:
		// { ln a a-ctime && rm a-ctime; }
		//
		//
		// # A has ctime more recent than C.
		// set $(ls -c a c)
		// test "$*" = 'a c'
		//
		// # Sleep so long in an attempt to avoid spurious failures
		// # due to NFS caching and/or clock skew.
		// sleep 2
		//
		// # Create a link, updating c's ctime.
		// ln c d
		//
		// # Before we go any further, verify that touch's -m option works.
		// set -- $(ls --full -l --time=mtime a)
		// case "$*" in
		//   *" $t3:00.000000000 +0000 a") ;;
		//   *)
		//   # This might be what's making HPUX 11 systems fail this test.
		//   cat >&2 << EOF
		// A basic test of touch -m has just failed, so the subsequent
		// tests in this file will not be run.
		//
		// In the output below, the date of last modification for 'a' should
		// have been $t3.
		// EOF
		//   ls --full -l a
		//   skip_ "touch -m -d '$t3' didn't work"
		//   ;;
		// esac
		//
		// # Ensure that touch's -a option works.
		// set -- $(ls --full -lu a)
		// case "$*" in
		//   *" $u1:00.000000000 +0000 a") ;;
		//   *)
		//   # This might be what's making HPUX 11 systems fail this test.
		//   cat >&2 << EOF
		// A fundamental touch -a test has just failed, so the subsequent
		// tests in this file will not be run.
		//
		// In the output below, the date of last access for 'a' should
		// have been $u1.
		// EOF
		//   ls --full -lu a
		//   Exit 77
		//   ;;
		// esac
		//
		// set $(ls -ut a b c)
		// test "$*" = 'c b a' && :
		// test $fail = 1 && ls -l --full-time --time=access a b c
		//
		// set $(ls -t a b c)
		// test "$*" = 'a b c' && :
		// test $fail = 1 && ls -l --full-time a b c
		//
		// # Now, C should have ctime more recent than A.
		// set $(ls -ct a c)
		// if test "$*" = 'c a'; then
		//   : ok
		// else
		//   # In spite of documentation, (e.g., stat(2)), neither link nor chmod
		//   # update a file's st_ctime on SunOS4.1.4.
		//   cat >&2 << \EOF
		// failed ls ctime test -- this failure is expected at least for SunOS4.1.4
		// and for tmpfs file systems on Solaris 5.5.1.
		// It is also expected to fail on a btrfs file system until
		// https://bugzilla.redhat.com/591068 is addressed.
		//
		// In the output below, 'c' should have had a ctime more recent than
		// that of 'a', but does not.
		// EOF
		//   #'
		//   ls -ctl --full-time a c
		//   fail=1
		// fi
		//
		// # This check is ineffective if:
		// #   en_US locale is not on the system.
		// #   The system en_US message catalog has a specific TIME_FMT translation,
		// #   which was inadvertently the case between coreutils 8.1 and 8.5 inclusive.
		//
		// if gettext --version >/dev/null 2>&1; then
		//
		//   default_tf1='%b %e  %Y'
		//   en_tf1=$(LC_ALL=en_US gettext coreutils "$default_tf1")
		//
		//   if test "$default_tf1" = "$en_tf1"; then
		//     LC_ALL=en_US ls -l c >en_output
		//     ls -l --time-style=long-iso c >liso_output
		//     if compare en_output liso_output; then
		//       fail=1
		//       echo "Long ISO TIME_FMT being used for en_US locale." >&2
		//     fi
		//   fi
		// fi
	})

	t.Run("multihardlink", func(t *testing.T) {
		// Ensure "ls --color" properly colors names of hard linked files.
		start(t)

		// touch file file1
		// ln file1 file2 || skip_ "can't create hard link"
		// code_mh='44;37'
		// code_ex='01;32'
		// code_png='01;35'
		// c0=$(printf '\033[0m')
		// c_mh=$(printf '\033[%sm' $code_mh)
		// c_ex=$(printf '\033[%sm' $code_ex)
		// c_png=$(printf '\033[%sm' $code_png)
		//
		// # regular file - not hard linked
		// LS_COLORS="mh=$code_mh" ls -U1 --color=always file > out
		// printf "file\n" > out_ok
		// compare out out_ok
		//
		// # hard links
		// LS_COLORS="mh=$code_mh" ls -U1 --color=always file1 file2 > out
		// printf "$c0${c_mh}file1$c0
		// ${c_mh}file2$c0
		// " > out_ok
		// compare out out_ok
		//
		// # hard links and png (hard link coloring takes precedence)
		// mv file2 file2.png
		// LS_COLORS="mh=$code_mh:*.png=$code_png" ls -U1 --color=always file1 file2.png \
		//   > out
		// printf "$c0${c_mh}file1$c0
		// ${c_mh}file2.png$c0
		// " > out_ok
		// compare out out_ok
		//
		// # hard links and exe (exe coloring takes precedence)
		// chmod a+x file2.png
		// LS_COLORS="mh=$code_mh:*.png=$code_png:ex=$code_ex" \
		//   ls -U1 --color=always file1 file2.png > out
		// chmod a-x file2.png
		// printf "$c0${c_ex}file1$c0
		// ${c_ex}file2.png$c0
		// " > out_ok
		// compare out out_ok
		//
		// # hard links and png (hard link coloring disabled => png coloring enabled)
		// LS_COLORS="mh=00:*.png=$code_png" ls -U1 --color=always file1 file2.png > out \
		//
		// printf "file1
		// $c0${c_png}file2.png$c0
		// " > out_ok
		// compare out out_ok
		//
		// # hard links and png (hard link coloring not enabled explicitly => png coloring)
		// LS_COLORS="*.png=$code_png" ls -U1 --color=always file1 file2.png > out \
		//
		// printf "file1
		// $c0${c_png}file2.png$c0
		// " > out_ok
		// compare out out_ok
	})

	t.Run("nameless-uid", func(t *testing.T) {
		// Ensure that ls -l works on files with nameless uid and/or gid
		// require_root_
		// require_perl_
		start(t)

		// nameless_uid=$($PERL -e '
		//   foreach my $i (1000..16*1024) { getpwuid $i or (print "$i\n"), exit }
		// ')
		//
		// if test x$nameless_uid = x; then
		//   skip_ "couldn't find a nameless UID"
		// fi
		//
		// touch f
		// chown $nameless_uid f
		//
		//
		// set -- $(ls -o f)
		// test $3 = $nameless_uid
	})

	t.Run("no-arg", func(t *testing.T) {
		// make sure ls and 'ls -R' do the right thing when invoked with no arguments.
		start(t)

		mkdirAll(t, "dir/subdir")
		touch(t, "dir/subdir/file2")
		touch(t, "exp")
		touch(t, "out")
		symlink(t, "f", "symlink")

		{
			have := strings.Fields(mustRun(t, "-1"))
			want := []string{"dir", "exp", "out", "symlink"}
			if !reflect.DeepEqual(have, want) {
				t.Errorf("\nhave: %s\nwant: %s", have, want)
			}
		}

		have := mustRun(t, "-R1")
		want := strings.ReplaceAll(`
			.:
			dir
			exp
			out
			symlink

			dir:
			subdir

			dir/subdir:
			file2`[1:], "\t", "")
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
		}
	})

	t.Run("recursive", func(t *testing.T) {
		// 4.1.1 and 4.1.2 had a bug whereby some recursive listings didn't
		// include a blank line between per-directory groups of files.
		start(t)

		for _, d := range []string{"x", "y", "a", "b", "c", "a/1", "a/2", "a/3"} {
			mkdirAll(t, d)
		}
		for _, f := range []string{"f", "a/1/I", "a/1/II"} {
			touch(t, f)
		}

		// This first example is from Andreas Schwab's bug report.
		have := mustRun(t, "-R1", "a", "b", "c")
		want := strings.ReplaceAll(`
			a:
			1
			2
			3

			a/1:
			I
			II

			a/2:

			a/3:

			b:

			c:`[1:], "\t", "")
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
		}

		have = mustRun(t, "-R1", "x", "y", "f")
		want = strings.ReplaceAll(`
			f

			x:

			y:`[1:], "\t", "")
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
		}
	})

	t.Run("removed-directory", func(t *testing.T) {
		switch runtime.GOOS {
		case "illumos", "solaris", "windows":
			t.Skipf("can't delete used directory on %s", runtime.GOOS)
		case "netbsd":
			if isCI() {
				// helper_test.go:46: mustRun failed: elles-test: getwd: no such file or directory
				t.Skip("TODO: fails in CI")
			}
		}

		// If ls is asked to list a removed directory (e.g., the parent
		// process's current working directory has been removed by another
		// process), it should not emit an error message merely because the
		// directory is removed.
		start(t)
		mkdirAll(t, "d")
		cd(t, "d")
		rmAll(t, "../d")

		if have := mustRun(t); have != "" {
			t.Error(have)
		}
		if have, ok := run(t, "../d"); ok {
			t.Error(have)
		}
	})

	t.Run("root-rel-symlink-color", func(t *testing.T) {
		// 8.17 ls bug with coloring relative-named symlinks in "/".
		start(t)

		// symlink_to_rel=
		// for i in /*; do
		//   # Skip non-symlinks:
		//   env test -h "$i" || continue
		//
		//   # Skip dangling symlinks:
		//   env test -e "$i" || continue
		//
		//   # Skip any symlink-to-absolute-name:
		//   case $(readlink "$i") in /*) continue ;; esac
		//
		//   symlink_to_rel=$i
		//   break
		// done
		//
		// test -z "$symlink_to_rel" \
		//   && skip_ no relative symlink in /
		//
		// e='\33'
		// color_code='01;36'
		// c_pre="$e[0m$e[${color_code}m"
		// c_post="$e[0m"
		// printf "$c_pre$symlink_to_rel$c_post\n" > exp
		//
		// env TERM=xterm LS_COLORS="ln=$color_code:or=1;31;42" \
		//   ls -d --color=always "$symlink_to_rel" > out
		//
		// compare exp out
		//
		// Exit $fail
	})

	t.Run("rt-1", func(t *testing.T) { // Name is used as secondary key when sorting on time
		supportsUtimes(t, true)
		start(t)

		tt := time.Date(1998, 1, 15, 0, 0, 0, 0, time.Local)
		touchDate(t, tt, "c")
		touchDate(t, tt, "a")
		touchDate(t, tt, "b")

		{
			have := strings.Fields(mustRun(t, "-t"))
			if want := []string{"a", "b", "c"}; !reflect.DeepEqual(have, want) {
				t.Errorf("\nhave: %s\nwant: %s", have, want)
			}
		}
		{
			have := strings.Fields(mustRun(t, "-rt"))
			if want := []string{"c", "b", "a"}; !reflect.DeepEqual(have, want) {
				t.Errorf("\nhave: %s\nwant: %s", have, want)
			}
		}
	})

	t.Run("size-align", func(t *testing.T) { // test size alignment
		supportsSparseFiles(t, true)
		start(t)

		createSparse(t, 0, "small")
		createSparse(t, 123456, "large")
		echoAppend(t, "\n", "alloc")

		x := func(in string) string {
			out := make([]string, 0, 8)
			for _, l := range strings.Split(in, "\n") {
				x := strings.Split(l, "│")
				if len(x) < 3 {
					t.Errorf("unexpected:\n%q", l)
				}
				out = append(out, fmt.Sprintf("%s│%s", x[0], x[2]))
			}
			return strings.Join(out, "\n")
		}

		{
			have := x(mustRun(t, "-l"))
			want := strings.ReplaceAll(`
				    1 │ alloc
				 121K │ large
				    0 │ small`[1:], "\t", "")
			if have != want {
				t.Errorf("\nhave:\n%s\nwant:\n%s", have, want)
			}
		}

		{
			have := x(mustRun(t, "-l", "-Bs"))
			want := strings.ReplaceAll(`
				 8 │ alloc
				 0 │ large
				 0 │ small`[1:], "\t", "")
			if have != want {
				t.Errorf("\nhave:\n%s\nwant:\n%s", have, want)
			}
		}

		{
			have := x(mustRun(t, "-l", "-BS"))
			want := strings.ReplaceAll(`
				  1 │ alloc
				 31 │ large
				  0 │ small`[1:], "\t", "")
			if have != want {
				t.Errorf("\nhave:\n%s\nwant:\n%s", have, want)
			}
		}
	})

	t.Run("slink-acl", func(t *testing.T) {
		// verify that ls -lL works when applied to a symlink to an ACL'd file

		// require_setfacl_

		// touch k
		// setfacl -m user::r-- k
		// ln -s k s

		// set _ $(ls -Log s); shift; link=$1
		// set _ $(ls -og k);  shift; reg=$1

		// test "$link" = "$reg"
	})

	t.Run("sort-width-option", func(t *testing.T) { // --sort=width option.
		start(t)

		mkdirAll(t, "subdir")
		touch(t, "subdir/aaaaa")
		touch(t, "subdir/bbb")
		touch(t, "subdir/cccc")
		touch(t, "subdir/d")
		touch(t, "subdir/zz")

		have := strings.Fields(mustRun(t, "-W", "subdir"))
		want := []string{"d", "zz", "bbb", "cccc", "aaaaa"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	})

	t.Run("stat-dtype", func(t *testing.T) {
		// Ensure that ls --file-type does not call stat unnecessarily. Also
		// check for the dtype-related (and fs-type dependent) bug in
		// coreutils-6.0 that made ls -CF columns misaligned.
		//
		// The trick is to create an un-stat'able symlink and to see if ls can
		// report its type nonetheless, using dirent.d_type.
		//
		// Skip this test unless "." is on a file system with useful d_type
		// info.
		// FIXME: This uses "ls -p" to decide whether to test "ls" with other
		// options, but if ls's d_type code is buggy then "ls -p" might be buggy
		// too.

		// mkdir -p c/d
		// chmod a-x c
		// if test "X$(ls -p c 2>&1)" != Xd/; then
		//   skip_ "'.' is not on a suitable file system for this test"
		// fi

		// mkdir d
		// ln -s / d/s
		// chmod 600 d

		// mkdir -p e/a2345 e/b
		// chmod 600 e

		// ls --file-type d > out
		// cat <<\EOF > exp
		// s@
		// EOF
		// compare exp out

		// Check for the ls -CF misaligned-columns bug:
		// ls -CF e > out

		// coreutils-6.0 would print two spaces after the first slash,
		// rather than the appropriate TAB.
		// printf 'a2345/\tb/\n' > exp
		// compare exp out
	})

	t.Run("stat-failed", func(t *testing.T) {
		// Verify that ls works properly when it fails to stat a file that is
		// not mentioned on the command line.
		start(t)

		// LS_MINOR_PROBLEM=1
		//
		// mkdir d
		// ln -s / d/s
		// chmod 600 d

		// returns_ 1 ls -Log d > out
		//
		// # Linux 2.6.32 client with Isilon OneFS always returns d_type==DT_DIR ('d')
		// # Newer Linux 3.10.0 returns the more correct DT_UNKNOWN ('?')
		// grep '^[l?]?' out || skip_ 'unrecognized d_type returned'

		// cat <<\EOF > exp
		// total 0
		// ?????????? ? ?            ? s
		// EOF
		// sed 's/^l/?/' out | compare exp -
	})

	t.Run("symlink-loop", func(t *testing.T) { // ls symlink ELOOP handling
		if runtime.GOOS == "windows" {
			t.Skip()
		}
		start(t)
		symlink(t, "loop", "loop")

		loopError := func(s string) bool {
			return strings.Contains(s, "too many levels of symbolic links") ||
				strings.Contains(s, "exceeds MAXSYMLINKS")
		}

		if have := mustRun(t, "-1", "loop"); have != "loop" {
			t.Error(have)
		}
		if have := mustRun(t, "-1l", "loop"); strings.Count(have, "\n") != 0 || !strings.Contains(have, "loop → loop") {
			t.Error(have)
		}
		if have, ok := run(t, "-1H", "loop"); ok || !loopError(have) {
			t.Error(ok, have)
		}
		if runtime.GOOS == "darwin" {
			// TODO: on macOS it exits with 0
			if have, ok := run(t, "-l1", "loop/"); !loopError(have) {
				t.Errorf("%v\n%s", ok, have)
			}
		} else {
			if have, ok := run(t, "-l1", "loop/"); ok || !loopError(have) {
				t.Errorf("%v\n%s", ok, have)
			}
		}
	})

	t.Run("symlink-slash", func(t *testing.T) { // Dereference symlink arg if written with a trailing slash.
		start(t)
		mkdirAll(t, "dir")
		touch(t, "dir/inside")
		symlink(t, "dir", "symlink")

		if have := mustRun(t, "-1", "symlink"); have != "symlink" {
			t.Error(have)
		}
		if have := mustRun(t, "-1", "symlink/"); have != "inside" {
			t.Error(have)
		}
	})
}
