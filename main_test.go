package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"zgo.at/elles/os2"
)

// Includes tests converted from FreeBSD (commit 0dfd11abc) and GNU coreutils
// (commit bbc972b).

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
	// but Go doesn't expose that.
	lnkprm, lnkprmO := "lrwxr-xr-x", " 755"
	switch runtime.GOOS {
	case "linux", "illumos", "solaris":
		lnkprm, lnkprmO = "lrwxrwxrwx", " 777"
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

	t.Run("-o", func(t *testing.T) {
		have := mustRun(t, "-llgo")
		want := norm(`
			 644 martin tournoij 1.0M Jan _2 15:04 │ 1M
			 644 martin tournoij    0 Jan _2 15:04 │ dir
			 644 martin tournoij 9.8K Jan _2 15:04 │ file
			`+lnkprmO+` martin tournoij    3 Jan _2 15:04 │ ln-dir → dir
			`+lnkprmO+` martin tournoij    4 Jan _2 15:04 │ ln-file → file`,
			"Jan _2 15:04", now.Format("Jan _2 15:04"))
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	})
}

func TestLongSizeAlignment(t *testing.T) {
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
}

func TestSortMtime(t *testing.T) {
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
}

func TestSortAtime(t *testing.T) {
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

// Name is used as secondary key when sorting on time
func TestSortTimeName(t *testing.T) {
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

		//{"000", "00", "01", "010", "09", "0", "1", "9", "10"},
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

func TestSortWidth(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip() // Doesn't like the \n
	}
	start(t)

	mkdirAll(t, "subdir")
	touch(t, "subdir/aaaaa")
	touch(t, "subdir/bbb")
	touch(t, "subdir/cccc")
	touch(t, "subdir/d")
	touch(t, "subdir/zz")
	touch(t, "subdir/€")
	touch(t, "subdir/a\nb")

	have := strings.Fields(mustRun(t, "-W", "subdir"))
	want := []string{"d", "€", "zz", "a$'\\n'b", "bbb", "cccc", "aaaaa"}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("\nhave: %s\nwant: %s", have, want)
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

func TestLFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	testFixedSizeWidth = true
	defer func() { testFixedSizeWidth = false }()
	tmp := start(t)

	now := time.Now().Format("15:04")
	mkdirAll(t, "dir")
	touch(t, "dir/file")
	echoTrunc(t, strings.Repeat("a", 6666), "file-1")
	touch(t, "file-2")
	symlink(t, "file-1", "link-file-1")
	symlink(t, "file-2", "link-file-2")
	symlink(t, "dir", "link-dir")
	st, err := os.Stat("dir")
	if err != nil {
		t.Fatal(err)
	}
	dsz, _ := listSize(st, "", "", false)
	repl := []string{"21:35", now, "TMPDIR", tmp, "DIRSZ", fmt.Sprintf("%5s", dsz)}

	{
		have := mustRun(t, "-CL")
		want := "dir  file-1  file-2  link-dir  link-file-1  link-file-2"
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have := mustRun(t, "-lL")
		want := norm(`
			 DIRSZ │ 21:35 │ dir
			  6.5K │ 21:35 │ file-1
			     0 │ 21:35 │ file-2
			 DIRSZ │ 21:35 │ link-dir
			  6.5K │ 21:35 │ link-file-1
			     0 │ 21:35 │ link-file-2`,
			repl...)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{ // Make sure we sort by correct size
		have := mustRun(t, "-lLS")
		want := norm(`
			  6.5K │ 21:35 │ file-1
			  6.5K │ 21:35 │ link-file-1
			 DIRSZ │ 21:35 │ dir
			 DIRSZ │ 21:35 │ link-dir
			     0 │ 21:35 │ file-2
			     0 │ 21:35 │ link-file-2`,
			repl...)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{ // And width
		have := mustRun(t, "-lLW")
		want := norm(`
			 DIRSZ │ 21:35 │ dir
			  6.5K │ 21:35 │ file-1
			     0 │ 21:35 │ file-2
			 DIRSZ │ 21:35 │ link-dir
			  6.5K │ 21:35 │ link-file-1
			     0 │ 21:35 │ link-file-2`,
			repl...)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	t.Run("orphan", func(t *testing.T) {
		symlink(t, "orphan", "link-orphan")
		defer rm(t, "link-orphan")

		{
			have := mustRun(t, "-CL")
			want := "dir  file-1  file-2  link-dir  link-file-1  link-file-2  link-orphan"
			if have != want {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}
		{
			have, ok := run(t, "-CFL")
			if ok {
				t.Error("exit 0")
			}
			want := norm(`
				dir/  file-1  file-2  link-dir/  link-file-1  link-file-2  link-orphan
				elles: stat TMPDIR/link-orphan: no such file or directory`,
				repl...)
			if have != want {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}

		{
			have, ok := run(t, "-lL")
			if ok {
				t.Error("exit 0")
			}
			want := norm(`
				 DIRSZ │      21:35 │ dir
				  6.5K │      21:35 │ file-1
				     0 │      21:35 │ file-2
				 DIRSZ │      21:35 │ link-dir
				  6.5K │      21:35 │ link-file-1
				     0 │      21:35 │ link-file-2
				   ??? │ ????-??-?? │ link-orphan
				elles: stat TMPDIR/link-orphan: no such file or directory`,
				repl...)
			if have != want {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}
	})

	t.Run("loop", func(t *testing.T) {
		msg := `too many levels of symbolic links`
		if runtime.GOOS == "solaris" || runtime.GOOS == "illumos" {
			msg = `number of symbolic links encountered during path name traversal exceeds MAXSYMLINKS`
		}

		symlink(t, "link-loop", "link-loop")
		defer rm(t, "link-loop")

		{
			have := mustRun(t, "-CL")
			want := "dir  file-1  file-2  link-dir  link-file-1  link-file-2  link-loop"
			if have != want {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}
		{
			have, ok := run(t, "-CFL")
			if ok {
				t.Error("exit 0")
			}
			want := norm(`
				dir/  file-1  file-2  link-dir/  link-file-1  link-file-2  link-loop
				elles: stat TMPDIR/link-loop: ERRMSG`,
				append([]string{"ERRMSG", msg}, repl...)...)
			if have != want {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}

		{
			have, ok := run(t, "-lL")
			if ok {
				t.Error("exit 0")
			}
			want := norm(`
				 DIRSZ │      21:35 │ dir
				  6.5K │      21:35 │ file-1
				     0 │      21:35 │ file-2
				 DIRSZ │      21:35 │ link-dir
				  6.5K │      21:35 │ link-file-1
				     0 │      21:35 │ link-file-2
				   ??? │ ????-??-?? │ link-loop
				elles: stat TMPDIR/link-loop: ERRMSG`,
				append([]string{"ERRMSG", msg}, repl...)...)
			if have != want {
				t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
			}
		}
	})
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
		have := mustRun(t, "-lw10", "-w21")
		want := norm(`
		 0 │ 01:08 │ 0123456…
		 0 │ 01:08 │ 0123456…`, "01:08", now)
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

func TestRecurse(t *testing.T) {
	start(t)

	for _, d := range []string{"x", "y", "a", "b", "c", "a/1", "a/2", "a/3"} {
		mkdirAll(t, d)
	}
	for _, f := range []string{"f", "a/1/I", "a/1/II"} {
		touch(t, f)
	}

	// This first example is from Andreas Schwab's bug report.
	have := mustRun(t, "-R1", "a", "b", "c")
	want := norm(`
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

		c:`)
	if have != want {
		t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
	}

	have = mustRun(t, "-R1", "x", "y", "f")
	want = norm(`
		f

		x:

		y:`)
	if have != want {
		t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
	}
}

func TestRemovedDirectory(t *testing.T) {
	switch runtime.GOOS {
	case "illumos", "solaris", "windows":
		t.Skipf("can't delete used directory on %s", runtime.GOOS)
	case "netbsd":
		if isCI() {
			// helper_test.go:46: mustRun failed: elles: getwd: no such file or directory
			t.Skip("TODO: fails in CI")
		}
	}

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
}

func TestCase(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skipf("%s doesn't like two pathnames differing only in casing", runtime.GOOS)
	}

	start(t)

	for _, f := range []string{"aa", "AA", "aA", "Aa"} {
		touch(t, f)
	}

	have := mustRun(t, "-C")
	want := norm(`AA  Aa  aA  aa`)
	if have != want {
		t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
	}
}

func TestColumns(t *testing.T) {
	defer func() { columns = 80 }()
	columns = 40

	start(t)

	for _, f := range []string{"c", "d", "e", "i", "klmn", "opqr",
		"stuv", "wxyz", "xxxx", "Hello", "AA", "with space"} {
		touch(t, f)
	}
	mkdirAll(t, "dir")

	have := mustRun(t, "-C")
	want := norm(`
		AA     d    i     stuv        xxxx
		Hello  dir  klmn  with space
		c      e    opqr  wxyz`)
	if have != want {
		t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
	}
}

func TestSpace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows doesn't like filenames as just a space, or something")
	}

	start(t)
	for _, f := range []string{
		"with space",
		" leading space",
		"trailing space ",
		" ",
		"  ",
		"\t",
	} {
		touch(t, f)
	}

	{
		have := mustRun(t, "-1")
		want := norm(`
			$'\t'
			" "
			"  "
			" leading space"
			"trailing space "
			with space`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
	{
		have := mustRun(t, "-1Q")
		want := norm(`
			"\t"
			" "
			"  "
			" leading space"
			"trailing space "
			"with space"`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

func TestControlChar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("control characters aren't permitted in Windows")
	}

	start(t)

	for n := range byte(0x1e) {
		touch(t, string([]byte{n + 1}))
	}
	for n := range rune(24) {
		mkdirAll(t, string([]rune{n + 0x80}))
	}
	touch(t, string([]byte{0x7f}))
	touch(t, "a\x01b\x02")
	symlink(t, "a\x01b\x02", "link1")
	symlink(t, "\n", "link2")
	symlink(t, "\x7f", "link3")
	symlink(t, "\u0081", "link4")
	symlink(t, "link1", "link\x01b\x02")

	{
		have := mustRun(t, "-CF")
		want := norm(`
			$'\x01'  $'\n'    $'\x13'  $'\x1c'               $'\x7f'   $'\x88'/  $'\x91'/
			$'\x02'  $'\x0b'  $'\x14'  $'\x1d'               $'\x80'/  $'\x89'/  $'\x92'/
			$'\x03'  $'\x0c'  $'\x15'  $'\x1e'               $'\x81'/  $'\x8a'/  $'\x93'/
			$'\x04'  $'\r'    $'\x16'  a$'\x01'b$'\x02'      $'\x82'/  $'\x8b'/  $'\x94'/
			$'\x05'  $'\x0e'  $'\x17'  link$'\x01'b$'\x02'@  $'\x83'/  $'\x8c'/  $'\x95'/
			$'\x06'  $'\x0f'  $'\x18'  link1@                $'\x84'/  $'\x8d'/  $'\x96'/
			$'\x07'  $'\x10'  $'\x19'  link2@                $'\x85'/  $'\x8e'/  $'\x97'/
			$'\x08'  $'\x11'  $'\x1a'  link3@                $'\x86'/  $'\x8f'/
			$'\t'    $'\x12'  $'\e'    link4@                $'\x87'/  $'\x90'/`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		h := strings.Split(mustRun(t, "-l", "link1", "link2", "link3", "link4", "link\x01b\x02"), "\n")
		for i := range h {
			x := strings.Split(h[i], " │ ")
			h[i] = x[2]
		}
		have := strings.Join(h, "\n")
		want := norm(`
			link$'\x01'b$'\x02' → link1
			link1 → a$'\x01'b$'\x02'
			link2 → $'\n'
			link3 → $'\x7f'
			link4 → $'\x81'`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

func TestUnprintable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows doesn't like some of these")
	}

	start(t)

	for _, n := range []string{
		"A→B",
		"A\u200dB", // Zero-width joiner
		"A\u200eB", // Left-to-right mark
		"A\u202dB", // Left-to-right override
		"A\ufe0eB", // text variation selector
		"A\ufe0fB", // emoji variation selector
		"A\ufe04B", // Mongolian variation selector
		"a\u0305b", // Combining overline
	} {
		touch(t, n)
	}

	{
		have := mustRun(t, "-C")
		want := norm(`
			A$'\u200d'B  A$'\u202d'B  A$'\ufe04'B  A$'\ufe0f'B
			A$'\u200e'B  A→B          A$'\ufe0e'B  a̅b`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have := mustRun(t, "-CQ")
		want := norm(`
			"A\u200dB"  "A\u202dB"  "A\ufe04B"  "A\ufe0fB"
			"A\u200eB"  A→B         "A\ufe0eB"  a̅b`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

func TestSpecialShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("many of these are not valid on Windows")
	}

	start(t)
	for _, n := range []string{
		"~", "`", "!", "#", "$", "%", "&", "*", "(", ")",
		"[", "]", "{", "}", "|", "\\", ";", ":", `"`, "'",
		",", ">", "<", "?",
		"...",
	} {
		touch(t, n)
	}

	{
		have := mustRun(t, "-aC")
		want := "!  \"  #  $  %  &  '  (  )  *  ,  ...  :  ;  <  >  ?  [  \\  ]  `  {  |  }  ~"
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have := mustRun(t, "-aQC")
		want := norm(`
			"!"   "#"  "%"  "'"  ")"  ,    :    "<"  "?"  "\"  "\` + "`" + `"  "|"  "~"
			"\""  "$"  "&"  "("  "*"  ...  ";"  ">"  "["  "]"  "{"   "}"`) // "
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

func TestDirFlag(t *testing.T) {
	start(t)
	mkdirAll(t, "a/b")

	for _, p := range []string{".", pwd(t), "a"} {
		o := mustRun(t, "-1d", p)
		if o != p {
			t.Fatalf("output not equal:\npath: %q\nout:  %q", p, o)
		}
	}
}

func TestGroupDirs(t *testing.T) {
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
}

// Make sure 'ls' and 'ls -R' do the right thing when invoked with no arguments.
func TestWithoutArguments(t *testing.T) {
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
	want := norm(`
		.:
		dir
		exp
		out
		symlink

		dir:
		subdir

		dir/subdir:
		file2`)
	if have != want {
		t.Errorf("\nhave:\n%s\n\nwant:\n%s\n\nhave: %[1]q\nwant: %[2]q", have, want)
	}
}

func TestSymlinkLoop(t *testing.T) {
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
		// TODO: on macOS it exits with 0 and:
		//  4 │ 22:34 │ loop/loop → ???
		t.Skip()
		// if have, ok := run(t, "-l1", "loop/"); !loopError(have) {
		// 	t.Errorf("%v\n%s", ok, have)
		// }
	} else {
		if have, ok := run(t, "-l1", "loop/"); ok || !loopError(have) {
			t.Errorf("%v\n%s", ok, have)
		}
	}
}

// Dereference symlink arg if written with a trailing slash.
func TestSymlinkSlash(t *testing.T) {
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
}

// Verify that ls works properly when it fails to stat a file that is not
// mentioned on the command line.
//
// This (indirectly) also tests whether just "elles" runs stat: GNU ls has a
// separate test for this, but if the first "elles dir" fails, then it
// (probably) ran stat when it shouldn't.
func TestUnreadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip() // TODO: figure out how to make directory unreadable on Windows.
	}
	if runtime.GOOS == "solaris" || runtime.GOOS == "illumos" {
		// TODO: outright fails with:
		//
		//   elles: lstat dir/link: permission denied
		//
		// That error comes from os2.ReadDir(a) in gather(); it seems either
		// os.Open() or os.File.ReadDir() call Stat on illumos/Solaris for some
		// reason(?) Need to investigate later.
		t.Skip()
	}

	start(t)
	defer chmod(t, 0o700, "dir")

	mkdirAll(t, "dir")
	symlink(t, "/", "dir/link")
	chmod(t, 0o600, "dir")

	{
		have := mustRun(t, "-C", "dir")
		want := `link`
		if have != want {
			t.Errorf("\nhave: %s\nwant: %s", have, want)
		}
	}

	{
		have, ok := run(t, "-CF", "dir")
		_ = ok
		want := norm(`
			link@
			elles: lstat dir/link: permission denied`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have, ok := run(t, "-l", "dir")
		_ = ok
		want := norm(`
			 ??? │ ????-??-?? │ link → ???
			elles: lstat dir/link: permission denied`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}

	{
		have, ok := run(t, "-ll", "dir")
		_ = ok
		want := norm(`
			l---------  :[failed] ??? ????-??-?? │ link → ???
			elles: lstat dir/link: permission denied`)
		if have != want {
			t.Errorf("\nhave:\n%s\n\nwant:\n%s", have, want)
		}
	}
}

// Ensure that ls -l works on files with nameless uid and/or gid
func TestNamelessUID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip() // TODO
	}

	hasRoot(t, true)

	var uid, gid int
	for i := range 16 * 1024 {
		i += 1000
		n := strconv.Itoa(i)
		if uid == 0 {
			_, err := user.LookupId(n)
			if err != nil {
				uid = i
			}
		}
		if gid == 0 {
			_, err := user.LookupGroupId(n)
			if err != nil && i != uid {
				gid = i
			}
		}
		if uid != 0 && gid != 0 {
			break
		}
	}
	if uid == 0 {
		t.Error("couldn't find a nameless UID")
	}
	if gid == 0 {
		t.Error("couldn't find a nameless GID")
	}

	start(t)
	touch(t, "file")
	mkdirAll(t, "dir")
	if err := os.Chown("file", uid, gid); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown("dir", uid, gid); err != nil {
		t.Fatal(err)
	}

	have := strings.Split(mustRun(t, "-ll"), "\n")
	want := fmt.Sprintf("%d :%d", uid, gid)
	for _, h := range have {
		if !strings.Contains(h, want) {
			t.Error(h)
		}
	}
}
