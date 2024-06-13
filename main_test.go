package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"zgo.at/elles/os2"
	"zgo.at/zli"
)

func TestGroupDigits(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"1", "1"},
		{"12", "12"},
		{"123", "123"},
		{"1234", "1,234"},
		{"123456", "123,456"},
		{"12345678", "12,345,678"},

		{"123.0", "123.0"},
		{"123.10", "123.10"},
		{"1234.0", "1,234.0"},
		{"1234.10", "1,234.10"},
		{"123456.0", "123,456.0"},
		{"123456.10", "123,456.10"},
		{"12345678.0", "12,345,678.0"},
		{"12345678.10", "12,345,678.10"},

		{"102G", "102G"},
		{"1024G", "1,024G"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			have := groupDigits(tt.in)
			if have != tt.want {
				t.Errorf("\nhave: %q\nwant: %q", have, tt.want)
			}
		})
	}
}

func BenchmarkGroupDigits(b *testing.B) {
	var g any
	b.Run("no suffix", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			g = groupDigits("12345678.10")
		}
	})
	b.Run("with suffix", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			g = groupDigits("12345678K")
		}
	})
	_ = g
}

func TestJSON(t *testing.T) {
	if isCI() || runtime.GOOS == "windows" {
		t.Skip("TODO")
	}

	start(t)
	touch(t, "file1")
	touch(t, "file2")

	var have, want []map[string]any
	err := json.Unmarshal([]byte(mustRun(t, "-j")), &have)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal([]byte(`
		[{
		  "abs_dir": "/tmp/TestJSON3123184094/001",
		  "dir":     ".",
		  "entries": [
		    {
		      "access_time": "2024-06-10T01:39:35.284680724+01:00",
		      "birth_time":  "2024-06-10T01:39:35.284680724+01:00",
		      "mod_time":    "2024-06-10T01:39:35.284680724+01:00",
		      "name":        "file1",
		      "permission":  420,
		      "size":        0,
		      "type":        0
		    },
		    {
		      "access_time": "2024-06-10T01:39:35.284680724+01:00",
		      "birth_time":  "2024-06-10T01:39:35.284680724+01:00",
		      "mod_time":    "2024-06-10T01:39:35.284680724+01:00",
		      "name":        "file2",
		      "permission":  420,
		      "size":        0,
		      "type":        0
		    }
		  ]
		}]`), &want)
	if err != nil {
		t.Fatal(err)
	}
	for i := range have {
		have[i]["abs_dir"] = want[i]["abs_dir"]
		for j := range have[i]["entries"].([]any) {
			m := have[i]["entries"].([]any)[j].(map[string]any)
			m["access_time"] = want[i]["entries"].([]any)[j].(map[string]any)["access_time"]
			m["birth_time"] = want[i]["entries"].([]any)[j].(map[string]any)["birth_time"]
			m["mod_time"] = want[i]["entries"].([]any)[j].(map[string]any)["mod_time"]
		}
	}
	if !reflect.DeepEqual(have, want) {
		h, _ := json.MarshalIndent(have, "", "  ")
		w, _ := json.MarshalIndent(want, "", "  ")
		t.Errorf("\nhave:\n%s\n\nwant:\n%s", h, w)
	}
}

func TestQuoteFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		// TODO: split to separate test.
		// Also: look up if quote characters are different on Windows.
		t.Skip("control characters aren't permitted in Windows")
	}

	start(t)
	for _, f := range []string{
		"\x01",
		"\n'",
		"\"dbl\"",
		"$",
		"'quote'",
		"(paren)",
		"**",
		"1M",
		">",
		"?",
		"Hello tab: \t lol",
		"[bracket]",
		"`",
		"bs \\",
		"file",
		"file with space",
		"€",
		"zwj: \u200d",
		"cancel tag: \U000e007f",
	} {
		touch(t, f)
	}

	{
		have := strings.Split(mustRun(t, "-1"), "\n")
		want := []string{
			"$'\\x01'",
			"$'\\n''",
			"\"dbl\"",
			"$",
			"'quote'",
			"(paren)",
			"**",
			"1M",
			">",
			"?",
			"Hello tab: $'\\t' lol",
			"[bracket]",
			"`",
			"bs \\",
			"cancel tag: $'\\U000e007f'",
			"file",
			"file with space",
			"zwj: $'\\u200d'",
			"€",
		}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}
	{
		have := strings.Split(mustRun(t, "-1Q"), "\n")
		want := []string{
			`"\x01"`,
			`"\n'"`,
			`"\"dbl\""`,
			`"$"`,
			`"'quote'"`,
			`"(paren)"`,
			`"**"`,
			`1M`,
			`">"`,
			`"?"`,
			`"Hello tab: \t lol"`,
			`"[bracket]"`,
			"\"\\`\"",
			`"bs \"`,
			`"cancel tag: \U000e007f"`,
			`file`,
			`"file with space"`,
			`"zwj: \u200d"`,
			`€`,
		}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}
	{
		have := strings.Split(mustRun(t, "-1QQ"), "\n")
		want := []string{
			`"\x01"`,
			`"\n'"`,
			`"\"dbl\""`,
			`"$"`,
			`"'quote'"`,
			`"(paren)"`,
			`"**"`,
			`"1M"`,
			`">"`,
			`"?"`,
			`"Hello tab: \t lol"`,
			`"[bracket]"`,
			"\"\\`\"",
			`"bs \"`,
			`"cancel tag: \U000e007f"`,
			`"file"`,
			`"file with space"`,
			`"zwj: \u200d"`,
			`"€"`,
		}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}
}

func TestLong(t *testing.T) {
	supportsUtimes(t, true)

	start(t)
	echoTrunc(t, strings.Repeat("x", 9999), "file")
	echoTrunc(t, strings.Repeat("x", 1024*1024+6), "1M")
	touch(t, "dir")
	symlink(t, "file", "ln-file")
	symlink(t, "dir", "ln-dir")
	now := time.Now()

	t.Run("default", func(t *testing.T) {
		have := mustRun(t, "-l")
		want := norm(`
			 1.0M │ 15:04 │ 1M
			    0 │ 15:04 │ dir
			 9.8K │ 15:04 │ file
			    3 │ 15:04 │ ln-dir → dir
			    4 │ 15:04 │ ln-file → file`,
			"15:04", now.Format("15:04"))
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	})
	t.Run("-T", func(t *testing.T) {
		have := mustRun(t, "-lT")
		want := norm(`
			 1.0M │ 2006-01-02 15:04:05 │ 1M
			    0 │ 2006-01-02 15:04:05 │ dir
			 9.8K │ 2006-01-02 15:04:05 │ file
			    3 │ 2006-01-02 15:04:05 │ ln-dir → dir
			    4 │ 2006-01-02 15:04:05 │ ln-file → file`,
			"2006-01-02 15:04:05", now.Format("2006-01-02 15:04:05"))
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	})
	t.Run("-TT", func(t *testing.T) {
		supportsUtimes(t, true)

		tt := time.Date(1985, 6, 16, 12, 13, 14, 15, time.Local)
		for _, f := range []string{"1M", "dir", "file", "ln-dir", "ln-file"} {
			if err := os2.Utimes(f, time.Time{}, tt); err != nil {
				t.Fatal(err)
			}
		}

		have := mustRun(t, "-lTT")
		want := norm(`
			 1.0M │ 2006-01-02 15:04:05.000000000 -07:00 │ 1M
			    0 │ 2006-01-02 15:04:05.000000000 -07:00 │ dir
			 9.8K │ 2006-01-02 15:04:05.000000000 -07:00 │ file
			    3 │ 2006-01-02 15:04:05.000000000 -07:00 │ ln-dir → dir
			    4 │ 2006-01-02 15:04:05.000000000 -07:00 │ ln-file → file`,
			"2006-01-02 15:04:05.000000000 -07:00", tt.Format("2006-01-02 15:04:05.000000000 -07:00"))
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	})
}

func TestLongLong(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO")
	}
	start(t)
	now := time.Now()

	echoTrunc(t, strings.Repeat("x", 9999), "file")
	echoTrunc(t, strings.Repeat("x", 1024*1024+6), "1M")
	touch(t, "dir")
	symlink(t, "file", "ln-file")
	symlink(t, "dir", "ln-dir")
	for _, f := range []string{"file", "1M", "dir", "ln-file", "ln-dir"} {
		os.Lchown(f, userinfo.UID, userinfo.GID)
	}

	// Permissions are different on Linux and BSD :-/ Can lchown() them on BSD,
	// but Go doesn't expoet that.
	lnkprm := "lrwxr-xr-x"
	switch runtime.GOOS {
	case "linux", "illumos", "solaris":
		lnkprm = "lrwxrwxrwx"
	}

	t.Run("default", func(t *testing.T) {
		have := mustRun(t, "-llg")
		want := norm(`
			-rw-r--r-- martin tournoij 1.0M Jan _2 15:04 │ 1M
			-rw-r--r-- martin tournoij    0 Jan _2 15:04 │ dir
			-rw-r--r-- martin tournoij 9.8K Jan _2 15:04 │ file
			`+lnkprm+` martin tournoij    3 Jan _2 15:04 │ ln-dir → dir
			`+lnkprm+` martin tournoij    4 Jan _2 15:04 │ ln-file → file`,
			"Jan _2 15:04", now.Format("Jan _2 15:04"))
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	})

	t.Run("-T", func(t *testing.T) {
		have := mustRun(t, "-llgT")
		want := norm(`
			-rw-r--r-- martin tournoij 1.0M 2006-01-02 15:04:05 │ 1M
			-rw-r--r-- martin tournoij    0 2006-01-02 15:04:05 │ dir
			-rw-r--r-- martin tournoij 9.8K 2006-01-02 15:04:05 │ file
			`+lnkprm+` martin tournoij    3 2006-01-02 15:04:05 │ ln-dir → dir
			`+lnkprm+` martin tournoij    4 2006-01-02 15:04:05 │ ln-file → file`,
			"2006-01-02 15:04:05", now.Format("2006-01-02 15:04:05"))
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	})

	t.Run("-n", func(t *testing.T) {
		have := mustRun(t, "-llgn")
		want := norm(`
			-rw-r--r-- XXXX YYYY 1.0M Jan _2 15:04 │ 1M
			-rw-r--r-- XXXX YYYY    0 Jan _2 15:04 │ dir
			-rw-r--r-- XXXX YYYY 9.8K Jan _2 15:04 │ file
			`+lnkprm+` XXXX YYYY    3 Jan _2 15:04 │ ln-dir → dir
			`+lnkprm+` XXXX YYYY    4 Jan _2 15:04 │ ln-file → file`,
			"Jan _2 15:04", now.Format("Jan _2 15:04"),
			"XXXX", userinfo.UserID,
			"YYYY", userinfo.GroupID)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	})
}

func TestSortBtime(t *testing.T) {
	supportsBtime(t, true)

	start(t)
	for _, f := range []string{"z", "a", "b"} {
		touch(t, f)
		time.Sleep(10 * time.Millisecond)
	}

	{
		have := strings.Fields(mustRun(t, "-tc"))
		want := []string{"b", "a", "z"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}
	{
		echoTrunc(t, "a", "a") // Shouldn't affect anything.
		have := strings.Fields(mustRun(t, "-tcr"))
		want := []string{"z", "a", "b"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}
}

func TestSortSize(t *testing.T) {
	start(t)

	createSparse(t, 0, "aaa")
	createSparse(t, 1, "bbb")
	createSparse(t, 1, "qqq")
	createSparse(t, 0, "yyy")
	createSparse(t, 2, "zzz")

	have := strings.Split(mustRun(t, "-1S"), "\n")
	want := strings.Fields(`zzz bbb qqq aaa yyy`)
	if !reflect.DeepEqual(have, want) {
		t.Errorf("\nhave: %s\nwant: %s", have, want)
	}
}

func TestSortExt(t *testing.T) {
	start(t)
	for _, f := range []string{"none", "file.txt", "zz.png", "img.png", "a.png"} {
		touch(t, f)
	}

	{
		have := strings.Fields(mustRun(t, "-X"))
		want := []string{"none", "a.png", "img.png", "zz.png", "file.txt"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}
	{
		have := strings.Fields(mustRun(t, "-Xr"))
		want := []string{"file.txt", "zz.png", "img.png", "a.png", "none"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}
}

func TestSortVersion(t *testing.T) {
	tests := [][]string{
		{},
		{"0"},
		{"0", "1"},
		{"00", "02", "10"},
		{"0", "2", "10"},
		{"a", "z"},
		{"a2", "z100"},
		{"2b", "100a"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			start(t)
			for _, f := range tt {
				touch(t, f)
			}

			have := strings.Fields(mustRun(t, "-v"))
			if !reflect.DeepEqual(have, tt) {
				t.Errorf("\nhave: %s\nwant: %s", have, tt)
			}
		})
	}
}

func TestPathNames(t *testing.T) {
	start(t)
	mkdirAll(t, "dir-one")
	mkdirAll(t, "dir-two")
	touch(t, "file")
	touch(t, "dir-one/file1")
	touch(t, "dir-two/file2")

	have := strings.Fields(mustRun(t, "dir-one/file1", "dir-two/file2", "file"))
	want := []string{"dir-one/file", "dir-two/file1", "file2"}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("\nhave: %s\nwant: %s", have, want)
	}
}

func TestInode(t *testing.T) {
	supportsUtimes(t, true)

	if runtime.GOOS == "netbsd" && isCI() {
		t.Skip("dirsize")
	}

	start(t)
	touch(t, "file")
	mkdirAll(t, "dir")

	tt := time.Date(2023, 6, 11, 15, 05, 0, 0, time.Local)
	inodes := make([]string, 0, 2)
	for _, f := range []string{"dir", "file"} {
		st, err := os.Stat(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := os2.Utimes(f, tt, tt); err != nil {
			t.Fatal(err)
		}
		os.Lchown(f, userinfo.UID, userinfo.GID)
		inodes = append(inodes, fmt.Sprintf("%d", os2.Serial(".", st)))
	}

	{
		have := strings.Fields(mustRun(t, "-iC"))
		want := []string{inodes[0], "dir", inodes[1], "file"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\nwant:\n%s", have, want)
		}
	}

	if len(inodes[0]) > len(inodes[1]) {
		inodes[1] = strings.Repeat(" ", len(inodes[0])-len(inodes[1])) + inodes[1]
	} else {
		inodes[0] = strings.Repeat(" ", len(inodes[1])-len(inodes[0])) + inodes[0]
	}

	{
		have := mustRun(t, "-gliBS")
		want := norm(`
			XXX │ 1 │ 2023-06-11 │ dir
			YYY │ 0 │ 2023-06-11 │ file`,
			"XXX", inodes[0],
			"YYY", inodes[1],
		)
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\nwant:\n%s", have, want)
		}
	}

	{
		have := mustRun(t, "-glliBS")
		want := norm(`
			XXX drwxr-xr-x martin tournoij 1 Jun 11 15:05 │ dir
			YYY -rw-r--r-- martin tournoij 0 Jun 11 15:05 │ file`,
			"XXX", inodes[0],
			"YYY", inodes[1])
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\nwant:\n%s", have, want)
		}
	}
}

func TestAllFlag(t *testing.T) {
	start(t)

	// No files to report.
	if have := mustRun(t, "-a"); have != "" {
		t.Fatalf("%q", have)
	}
	mkdirAll(t, "d")
	if have := mustRun(t, "-a", "d"); have != "" {
		t.Fatalf("%q", have)
	}

	touch(t, ".file")
	mkdirAll(t, ".dir")
	touch(t, "d/.file2")
	mkdirAll(t, "d/.dir2")

	{
		have := strings.Fields(mustRun(t, "-a"))
		want := []string{".dir", ".file", "d"}
		if !reflect.DeepEqual(have, want) {
			t.Fatalf("\nhave: %s\nwant: %s", have, want)
		}
	}

	{
		have := strings.Fields(mustRun(t, "-a", "d"))
		want := []string{".dir2", ".file2"}
		if !reflect.DeepEqual(have, want) {
			t.Fatalf("\nhave: %s\nwant: %s", have, want)
		}
	}
}

func TestClassifyFlag(t *testing.T) {
	start(t)

	check := func(want ...string) {
		t.Helper()
		if runtime.GOOS == "windows" { // No executable files on Windows.
			for i := range want {
				want[i] = strings.TrimSuffix(want[i], "*")
			}
		}

		wantNoF := make([]string, len(want))
		for i := range want {
			wantNoF[i] = strings.TrimRight(want[i], `*/@|=`)
		}
		wantP := make([]string, len(want))
		for i := range want {
			wantP[i] = strings.TrimRight(want[i], `*@|=`)
		}

		if have := strings.Split(mustRun(t, "-1F"), "\n"); !reflect.DeepEqual(have, want) {
			t.Errorf("-1F:\nhave: %s\nwant: %s", have, want)
		}
		if have := strings.Split(mustRun(t, "-1p"), "\n"); !reflect.DeepEqual(have, wantP) {
			t.Errorf("-1p:\nhave: %s\nwant: %s", have, wantP)
		}
		if have := strings.Split(mustRun(t, "-1"), "\n"); !reflect.DeepEqual(have, wantNoF) {
			t.Errorf("-1\nhave: %s\nwant: %s", have, wantNoF)
		}
	}

	mkdirAll(t, "dir")
	touch(t, "regular")
	touch(t, "executable")
	chmod(t, 0o555, "executable")
	symlink(t, "regular", "slink-reg")
	symlink(t, "dir", "slink-dir")
	symlink(t, "nowhere", "slink-dangle")

	check("dir/",
		"executable*",
		"regular",
		"slink-dangle@",
		"slink-dir@",
		"slink-reg@")

	t.Run("fifo", func(t *testing.T) {
		supportsFIFO(t, true)
		l, err := net.Listen("unix", "socket")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		mkfifo(t, "fifo")
		check("dir/",
			"executable*",
			"fifo|",
			"regular",
			"slink-dangle@",
			"slink-dir@",
			"slink-reg@",
			"socket=")
	})

	t.Run("device nodes", func(t *testing.T) {
		supportsDevice(t, true)
		mknod(t, 20, "block")
		mknod(t, 10, "char")

		check("block",
			"char",
			"dir/",
			"executable*",
			"fifo|",
			"regular",
			"slink-dangle@",
			"slink-dir@",
			"slink-reg@")
	})
}

func TestInodeFlag(t *testing.T) {
	start(t)

	touch(t, "file1")
	touch(t, "dir1")
	symlink(t, "file1", "link1")
	symlink(t, "nowhere", "link2")
	if supportsFIFO(t, false) {
		mkfifo(t, "fifo")
		l, err := net.Listen("unix", "socket")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
	}
	if supportsDevice(t, false) {
		mknod(t, 10, "device")
	}

	ls, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, f := range ls {
		fi, err := f.Info()
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, fmt.Sprintf("%d %s", os2.Serial(".", fi), mustRun(t, "-1d", f.Name())))
	}

	have := strings.Split(mustRun(t, "-1ai"), "\n")
	for i := range have {
		have[i] = strings.TrimLeft(have[i], " ")
	}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
	}

	t.Run("symlinks", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("inodes for links are the same on Windows")
		}
		mkdirAll(t, "links")
		cd(t, "links")
		touch(t, "f")
		symlink(t, "f", "slink")

		// When listed explicitly:
		have := strings.Fields(mustRun(t, "-i"))
		if len(have) != 4 {
			t.Fatalf("len %d", len(have))
		}
		// The inode numbers should differ.
		if have[0] == have[2] {
			t.Fatalf("%q == %q", have[0], have[2])
		}

		// With -H, they must be the same, but only when explicitly listed.
		have = strings.Fields(mustRun(t, "-iH"))
		if have[0] == have[2] {
			t.Fatalf("%q != %q", have[0], have[2])
		}
		have = strings.Fields(mustRun(t, "-iH", "f", "slink"))
		if have[0] != have[2] {
			t.Fatalf("%q != %q", have[0], have[2])
		}
	})
}

func TestHFlag(t *testing.T) {
	start(t)

	mkdirAll(t, "dir")
	touch(t, "dir/file")
	symlink(t, "dir", "link-dir")
	symlink(t, "orphan", "link-orphan")

	if have := mustRun(t, "-1", "link-dir"); have != "link-dir" {
		t.Fatal(have)
	}
	if have := mustRun(t, "-H1", "link-dir"); have != "file" {
		t.Fatal(have)
	}

	if have := mustRun(t, "-1", "link-orphan"); have != "link-orphan" {
		t.Fatal(have)
	}
	if have, ok := run(t, "-H1", "link-orphan"); ok {
		t.Fatal(have)
	}
}

func TestFilesizes(t *testing.T) {
	var (
		kb = int64(1024)
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)

	start(t)
	supportsSparseFiles(t, true)

	for _, sz := range []int64{1, 512, 2 * kb, 10 * kb, 512 * kb, mb, gb, tb} {
		createSparse(t, sz, fmt.Sprintf("%d.file", sz))
	}

	run := func(flags ...string) string {
		var h []string
		for _, line := range strings.Split(mustRun(t, flags...), "\n") {
			x := strings.Split(line, "│")
			h = append(h, fmt.Sprintf("%s│%s", x[0], x[2]))
		}
		return strings.Join(h, "\n")
	}
	{
		have := run("-l")
		want := norm(`
			     1 │ 1.file
			 10.0K │ 10240.file
			  1.0M │ 1048576.file
			  1.0G │ 1073741824.file
			 1024G │ 1099511627776.file
			  2.0K │ 2048.file
			   512 │ 512.file
			  512K │ 524288.file`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have := run("-l", "-B", "1")
		want := norm(`
		             1 │ 1.file
		         10240 │ 10240.file
		       1048576 │ 1048576.file
		    1073741824 │ 1073741824.file
		 1099511627776 │ 1099511627776.file
		          2048 │ 2048.file
		           512 │ 512.file
		        524288 │ 524288.file`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have := run("-l,", "-B", "1")
		want := norm(`
		                 1 │ 1.file
		            10,240 │ 10240.file
		         1,048,576 │ 1048576.file
		     1,073,741,824 │ 1073741824.file
		 1,099,511,627,776 │ 1099511627776.file
		             2,048 │ 2048.file
		               512 │ 512.file
		           524,288 │ 524288.file`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

func TestTrim(t *testing.T) {
	defer func() { columns = 80 }()

	start(t)

	long := strings.Repeat("0123456789", 10)
	now := time.Now().Format("15:04")
	touch(t, "0123456789")
	touch(t, long)

	{
		columns = 10
		have := strings.Fields(mustRun(t, "-1", "-trim"))
		want := []string{"0123456789", "012345678…"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		columns = 10
		have := strings.Fields(mustRun(t, "-C", "-trim"))
		want := []string{"0123456789", "012345678…"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		columns = 10
		have := strings.Fields(mustRun(t, "-C", "-trim"))
		want := []string{"0123456789", "012345678…"}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		columns = 20
		have := mustRun(t, "-l", "-trim")
		want := norm(`
		 0 │ 01:08 │ 012345…
		 0 │ 01:08 │ 012345…`, "01:08", now)
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		columns = 100
		have := strings.Fields(mustRun(t, "-1", "-trim"))
		want := []string{"0123456789", long}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		columns = 100
		have := strings.Fields(mustRun(t, "-C", "-trim"))
		want := []string{"0123456789", long}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

func TestWidth(t *testing.T) {
	defer func() { columns = 80 }()
	start(t)

	long := strings.Repeat("0123456789", 10)
	now := time.Now().Format("15:04")
	touch(t, "0123456789")
	touch(t, long)

	{
		have := mustRun(t, "-1w10")
		want := "0123456789\n012345678…"
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		have := mustRun(t, "-Cw10")
		want := "0123456789  012345678…"
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		have := mustRun(t, "-lw20")
		want := norm(`
		 0 │ 01:08 │ 012345…
		 0 │ 01:08 │ 012345…`, "01:08", now)
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have := mustRun(t, "-1w100")
		want := "0123456789\n" + long
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		columns = 200
		have := mustRun(t, "-Cw100")
		want := "0123456789  " + long
		if !reflect.DeepEqual(have, want) {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

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

func TestColor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip() // TODO: just because of the FIFO etc.
	}

	defer clearColors()

	os.Unsetenv("LS_COLORS")
	os.Unsetenv("LSCOLORS")

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
