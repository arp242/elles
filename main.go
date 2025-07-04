package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"zgo.at/elles/os2"
	"zgo.at/termtext"
	"zgo.at/zli"
)

var (
	isTerm  = func() bool { return zli.IsTerminal(os.Stdout.Fd()) }()
	columns = func() int {
		if c := os.Getenv("COLUMNS"); c != "" {
			if n, err := strconv.Atoi(c); err != nil && n > 0 {
				return n
			}

		}

		// On "| less", "| head", etc. we can get the width from stdin, but that
		// never works on Windows. So try both.
		n, _, err := zli.TerminalSize(os.Stdout.Fd())
		if err != nil {
			n, _, err = zli.TerminalSize(os.Stdin.Fd())
		}
		if err != nil {
			return 0
		}
		return n
	}()
)

type (
	printable struct {
		dir     string // Belongs in dir; can be empty.
		absdir  string
		isFiles bool
		fi      []fileInfo
	}
	fileInfo struct {
		fs.FileInfo
		filepath, filepathAbs string
	}
)

func main() {
	f := zli.NewFlags(os.Args)
	var (
		help         = f.Bool(false, "help")
		version      = f.Bool(false, "version")
		manpage      = f.Bool(false, "manpage")
		completion   = f.String("", "completion")
		all          = f.Bool(false, "a", "all", "A", "almost-all")
		asJSON       = f.Bool(false, "j", "json")
		list         = f.IntCounter(0, "l")
		prDir        = f.Bool(false, "d", "directory")
		one          = f.Bool(!isTerm, "1")
		cols         = f.Bool(isTerm, "C")
		hyperlink    = f.Optional().String("never", "hyperlink", "hyper")
		color        = f.Optional().String("auto", "color", "colour")
		colorBSD     = f.Bool(false, "G")
		sortReverse  = f.Bool(false, "r", "reverse")
		sortSize     = f.Bool(false, "S")
		sortTime     = f.Bool(false, "t")
		sortExt      = f.Bool(false, "X")
		sortVersion  = f.Bool(false, "v")
		sortWidth    = f.Bool(false, "W")
		sortNone     = f.Bool(false, "U")
		sortNoneAll  = f.Bool(false, "f")
		sortFlag     = f.String("name", "sort")
		dirsFirst    = f.Bool(false, "group-dir", "group-dirs", "group-directories", "group-directories-first")
		derefCmdline = f.Bool(false, "H")
		derefAll     = f.Bool(false, "L")
		recurse      = f.Bool(false, "R", "recursive")
		classify     = f.Bool(false, "F")
		dirSlash     = f.Bool(false, "p")
		numericUID   = f.Bool(false, "n")
		inode        = f.Bool(false, "i", "inode")
		blockSize    = f.String("h", "B", "block", "blocks", "block-size")
		_            = f.Bool(false, "h") // No-op
		sizeBlock    = f.Bool(false, "s", "size")
		timeCreate   = f.Bool(false, "c")
		timeAccess   = f.Bool(false, "u")
		comma        = f.Bool(false, ",")
		quote        = f.IntCounter(0, "Q")
		fullTime     = f.IntCounter(0, "T")
		width        = f.Int(0, "w", "width")
		trim         = f.Bool(false, "trim")
		noTrim       = f.Bool(false, "no-trim")
		octal        = f.Bool(false, "o", "octal")
		group        = f.Bool(false, "g", "groupname")
		minCols      = f.Int(0, "m", "min")
		noExt        = f.Bool(false, "e", "no-ext")
		dirSize      = f.Bool(false, "D", "dirsize")
	)
	zli.F(f.Parse(zli.AllowMultiple()))
	if colorBSD.Bool() && !color.Set() {
		*color.Pointer() = "always"
	}
	switch strings.ToLower(color.String()) {
	case "auto", "tty", "if-tty":
	case "never", "no", "none":
		zli.WantColor = false
	case "always", "yes", "force", "":
		zli.WantColor = true
	default:
		zli.Fatalf("invalid value for -color: %q", color)
	}
	setColor()
	if help.Bool() {
		fmt.Fprint(zli.Stdout, usage)
		return
	}
	if version.Bool() {
		zli.PrintVersion(false)
		return
	}
	if manpage.Bool() {
		fmt.Print(usage.Mandoc("elles", 1))
		return
	}
	if completion.Set() {
		switch shell := completion.String(); shell {
		case "zsh":
			fmt.Print(zsh)
		default:
			zli.Fatalf("no completion for %q", shell)
		}
		return
	}

	if noTrim.Set() {
		*trim.Pointer() = false
	}
	if sizeBlock.Bool() {
		*blockSize.Pointer() = "s"
	}

	doLink := false
	switch strings.ToLower(hyperlink.String()) {
	case "auto", "tty", "if-tty":
		doLink = zli.WantColor
	case "never", "no", "none":
	case "always", "yes", "force", "":
		doLink = true
	default:
		zli.Fatalf("invalid value for -hyperlink: %q", hyperlink)
	}

	nostat := list.Int() == 0 && !classify.Bool() && !inode.Bool() && !asJSON.Bool()
	switch {
	case sortNone.Bool():
		*sortFlag.Pointer() = "none"
	case sortNoneAll.Bool():
		*sortFlag.Pointer() = "none-all"
	case sortSize.Bool():
		*sortFlag.Pointer(), nostat = "size", false
		nostat = false
	case sortTime.Bool():
		*sortFlag.Pointer(), nostat = "time", false
	case sortVersion.Bool():
		*sortFlag.Pointer() = "version"
	case sortExt.Bool():
		*sortFlag.Pointer() = "ext"
	case sortWidth.Bool():
		*sortFlag.Pointer() = "width"
	}
	switch sortFlag.String() {
	case "name", "none", "none-all", "size", "time", "version", "ext", "extension", "width":
	default:
		zli.Fatalf("invalid value for -sort: %q", sortFlag.String())
	}
	timeField := "mtime"
	if timeCreate.Bool() {
		timeField = "btime"
	} else if timeAccess.Bool() {
		timeField = "atime"
	}

	if len(f.Args) == 0 {
		f.Args = []string{"."}
	}

	errs := &errGroup{MaxSize: 100}

	// Gather list to print.
	toPrint := gather(f.Args, errs, all.Bool(), recurse.Bool(), dirSize.Bool(),
		prDir.Bool(), derefCmdline.Bool(), derefAll.Bool(), nostat)

	// Order it.
	order(toPrint, sortFlag.String(), timeField, sortReverse.Bool(), dirsFirst.Bool(), dirSize.Bool())

	// Print as JSON.
	if asJSON.Bool() {
		printJSON(toPrint, errs)
		return
	}

	opt := opts{
		blockSize:   blockSize.String(),
		classify:    classify.Bool(),
		cols:        cols.Bool(),
		comma:       comma.Bool(),
		dirSlash:    dirSlash.Bool(),
		fullTime:    fullTime.Int(),
		group:       group.Bool(),
		hyperlink:   doLink,
		inode:       inode.Bool(),
		list:        list.Int(),
		numericUID:  numericUID.Bool(),
		octal:       octal.Bool(),
		one:         one.Bool(),
		quote:       quote.Int(),
		recurse:     recurse.Bool(),
		timeField:   timeField,
		trim:        trim.Bool(),
		maxColWidth: width.Int(),
		derefAll:    derefAll.Bool(),
		minCols:     minCols.Int(),
		noExt:       noExt.Bool(),
		dirSize:     dirSize.Bool(),
	}

	draw(toPrint, errs, opt, cols.Set())
}

func draw(toPrint []printable, errs *errGroup, opt opts, colsSet bool) {
	for i, p := range toPrint {
		// Print direcrory headers, but not when recursing with -D and there are
		// no directories.
		if len(toPrint) > 1 && p.dir != "" /*&& !(opt.dirsOnly && len(p.fi) == 0)*/ {
			if i > 0 {
				fmt.Fprintln(zli.Stdout)
			}
			fmt.Fprintln(zli.Stdout, filepath.ToSlash(filepath.Clean(p.dir))+":")
		}

		// Format for output in memory first. This makes alignment much easier
		// because we may or may not add things such as "/". Even with very
		// large directories it shouldn't take more than a few hundred K.
		cc := getCols(p, opt)

	refmt:
		var (
			fmtRows = make([]string, 0, len(cc.rows))
			widths  = make([]int, 0, len(cc.rows))
			longest int
			buf     strings.Builder
			w       int
		)
		for _, r := range cc.rows {
			buf.Reset()
			w = 0
			for i, c := range r {
				if i > 0 {
					buf.WriteString(" ")
					w++
				}

				if c.prop&borderToLeft != 0 {
					buf.WriteString("│ ")
					w += 2
				}
				if c.prop&alignNone != 0 {
					w += c.w
					buf.WriteString(c.s)
				} else if c.prop&alignLeft != 0 {
					pad := cc.longest[i] - c.w
					buf.WriteString(c.s)
					buf.WriteString(strings.Repeat(" ", pad))
					w += c.w + pad
				} else {
					pad := cc.longest[i] - c.w
					buf.WriteString(strings.Repeat(" ", pad))
					buf.WriteString(c.s)
					w += c.w + pad
				}
			}

			b := buf.String()
			if opt.maxColWidth > 0 && w > opt.maxColWidth {
				b = termtext.Slice(b, 0, opt.maxColWidth-1) + reset + "…"
				w = opt.maxColWidth
			}
			fmtRows, widths = append(fmtRows, b), append(widths, w)
			if w > longest {
				longest = w
			}
		}

	one:
		if (opt.one && !colsSet) || (opt.list > 0 && !colsSet) {
			for i, f := range fmtRows {
				if columns > 0 && opt.trim && widths[i] > columns {
					f = termtext.Slice(f, 0, columns-1) + reset + "…"
				}
				fmt.Fprintln(zli.Stdout, f)
			}
		} else {
			var (
				colwidths []int
				rows      [][]string
				pad       = 2
			)
			if opt.list > 0 {
				pad = 4
			}
			for i := range 200 {
				if i == 0 {
					continue
				}
				r, w := recol(fmtRows, widths, i, pad)
				if sum(w) > columns {
					if i <= 1 {
						rows, colwidths = r, w
					}
					break
				}
				rows, colwidths = r, w
			}
			if opt.minCols > 0 && len(colwidths) < opt.minCols {
				if opt.maxColWidth == 0 {
					opt.maxColWidth = longest - 1
				} else {
					opt.maxColWidth--
				}
				goto refmt
			}

			// Only space for one column; restart as if -1 was set. Saves some
			// special-fu here.
			if len(colwidths) == 1 {
				opt.one, colsSet = true, false
				goto one
			}

			for i, r := range rows {
				for j, c := range r {
					x := i + len(rows)*j
					if opt.list > 0 && j != len(r)-1 {
						fmt.Fprint(zli.Stdout, c, strings.Repeat(" ", colwidths[j]-widths[x]-2))
						fmt.Fprint(zli.Stdout, "┃ ")
					} else {
						fmt.Fprint(zli.Stdout, c)
						if j != len(r)-1 {
							fmt.Fprint(zli.Stdout, strings.Repeat(" ", colwidths[j]-widths[x]))
						}
					}
				}
				fmt.Fprintln(zli.Stdout)
			}
		}
	}

	// Print errors last, so they're more visible. ls does this at the top, and
	// it's easy to miss if pushed off the screen.
	for _, e := range errs.List() {
		zli.Errorf(e)
	}
	if errs.Len() > 0 {
		zli.Exit(1)
	}
}

func recol(paths []string, pathWidths []int, ncols, pad int) ([][]string, []int) {
	var (
		rows   = make([][]string, 0, 8)
		widths = make([]int, ncols)
		height = int(math.Ceil(float64(len(paths)) / float64(ncols)))
	)
	for i := range height {
		row := make([]string, 0, ncols)
		for c := range ncols {
			j := i + height*c
			if j > len(paths)-1 {
				break
			}

			l := pathWidths[j]
			if c < ncols-1 {
				l += pad
			}
			if l > widths[c] {
				widths[c] = l
			}
			row = append(row, paths[j])
		}
		rows = append(rows, row)
	}
	if i := slices.Index(widths, 0); i > -1 {
		widths = widths[:i]
	}
	return rows, widths
}

func sum(s []int) int {
	var n int
	for _, ss := range s {
		n += ss
	}
	return n
}

// Gather list of everything we want to print.
func gather(args []string, errs *errGroup, all, recurse, dirSize, prDir, derefCmd, derefAll, nostat bool) []printable {
	var (
		toPrint    = make([]printable, 0, 16)
		filesIndex = -1 // index in toPrint for individual files.
		stat       = os.Lstat
	)
	if derefCmd {
		stat = os.Stat
	}
	//cwd, err := os.Getwd()
	//errs.Append(err)

	var addArg func(string)
	addArg = func(a string) {
		fi, err := stat(a)
		if err != nil {
			if a == "." && errors.Is(err, os.ErrNotExist) {
				return
			}
			if errs.Append(err) {
				return
			}
		}

		if fi.IsDir() && !prDir { /// Directory.
			ls, err := os2.ReadDir(a)
			if err != nil {
				if a == "." && errors.Is(err, os.ErrNotExist) {
					return
				}
				errs.Append(err)
				return
			}

			d := a
			//if strings.TrimRight(d, "/") == "." {
			//	d = cwd
			//}
			if !filepath.IsAbs(d) {
				if d == "." || d == "./" {
					d = "."
				} else {
					d = string(append([]byte{'.', filepath.Separator}, d...))
				}
			}
			ad, err := filepath.Abs(d)
			errs.Append(err)
			pr := printable{
				dir:    d,
				absdir: ad,
				fi:     make([]fileInfo, 0, len(ls)),
			}
			var subdirs []string
			for _, l := range ls {
				if os2.Hidden(ad, l) && !all {
					continue
				}
				//if !l.IsDir() && dirsOnly { continue }

				// Don't call stat if we don't need to.
				if nostat {
					pr.fi = append(pr.fi, fileInfo{fakeFileInfo{l}, "", ""})
				} else {
					var fi fs.FileInfo
					if derefAll {
						fi, err = os.Stat(filepath.Join(ad, l.Name()))
					} else {
						fi, err = l.Info()
					}
					if errs.Append(err) {
						// Don't skip the entire file, just don't add stat info.
						pr.fi = append(pr.fi, fileInfo{fakeFileInfo{l}, "", ""})
					} else {
						if fi.IsDir() && dirSize {
							fi = &rdir{path: filepath.Join(ad, l.Name()), fi: fi}
						}
						pr.fi = append(pr.fi, fileInfo{fi, "", ""})
					}
				}

				if recurse && l.IsDir() {
					subdirs = append(subdirs, filepath.Join(d, l.Name()))
				}
			}
			toPrint = append(toPrint, pr)
			for _, s := range subdirs {
				addArg(s)
			}
		} else { /// Single file (or directory with -d).
			if prDir {
				a = strings.TrimRight(a, "/")
			}
			d := strings.TrimSuffix(a, fi.Name())
			ad, err := filepath.Abs(d)
			errs.Append(err)
			if fi.IsDir() && dirSize {
				fi = &rdir{path: filepath.Join(ad, fi.Name()), fi: fi}
			}

			if filesIndex == -1 {
				toPrint = append(toPrint, printable{
					dir:     d,
					absdir:  ad,
					isFiles: true,
					fi:      []fileInfo{{fi, d, ad}},
				})
				filesIndex = len(toPrint) - 1
			} else {
				toPrint[filesIndex].fi = append(toPrint[filesIndex].fi, fileInfo{fi, d, ad})
			}
		}
	}
	for _, a := range args {
		// Make sure "ls /" and "ls C:" work on Windows.
		if runtime.GOOS == "windows" {
			if a == "/" {
				wd, err := os.Getwd()
				if err == nil {
					a = filepath.VolumeName(wd) + `\`
				}
			} else if len(a) == 2 && a[1] == ':' {
				a += `\`
			}
		}
		addArg(a)
	}
	return toPrint
}

//func getEnv(name string) (string, bool) {
//	l, ok := os.LookupEnv(name)
//	if !ok {
//		return "", false
//	}
//	l = strings.SplitN(l, ".", 2)[0] // Remove ".UTF-8" or ".ASCII" encoding
//	if l == "" || l == "C" {         // We can't do anything with this.
//		return "", false
//	}
//	return l, true
//}

// Sort files.
func order(toPrint []printable, sortby, timeField string, reverse, dirsFirst, dirSize bool) {
	var (
		sorter func(a, b fileInfo) int
		// TODO: Hack for Linux btime, until we rewrite some of the stdlib stuff.
		sorter2 func(printable) func(a, b fileInfo) int

		nameSort = func(a, b fileInfo) int { return cmp.Compare(a.Name(), b.Name()) }
	)

	// var (
	// 	lang     language.Tag
	// 	haveLang bool
	// )
	// for _, v := range []string{"LC_COLLATE", "LC_ALL", "LANG"} {
	// 	if e, ok := getEnv(v); ok {
	// 		langs, _, err := language.ParseAcceptLanguage(e)
	// 		if err != nil || len(langs) == 0 {
	// 			zli.Errorf("invalid %s: %s", v, err)
	// 		}
	// 		lang, haveLang = langs[0], true
	// 		break
	// 	}
	// }
	//if haveLang {
	//	col := collate.New(lang, collate.WithCase)
	//	nameSort = func(a, b fs.FileInfo) int { return col.CompareString(a.Name(), b.Name()) }
	//}

	switch sortby {
	case "size":
		// Make sure we have consistent sorting for -S, and also sort "below" 0.
		sorter = func(a, b fileInfo) int {
			n1, n2 := a.Size(), b.Size()
			if (!dirSize && a.IsDir()) || a.Mode()&fs.ModeSymlink != 0 {
				n1 = -1
			}
			if (!dirSize && b.IsDir()) || b.Mode()&fs.ModeSymlink != 0 {
				n2 = -1
			}
			return cmp.Compare(n2, n1)
		}
	case "time":
		switch timeField {
		case "btime":
			sorter = nil
			sorter2 = func(p printable) func(a, b fileInfo) int {
				return func(a, b fileInfo) int { return os2.Btime(p.absdir, b).Compare(os2.Btime(p.absdir, a)) }
			}
		case "atime":
			sorter = func(a, b fileInfo) int { return os2.Atime(b).Compare(os2.Atime(a)) }
		default:
			sorter = func(a, b fileInfo) int { return b.ModTime().Compare(a.ModTime()) }
		}
	case "ext", "extension":
		sorter = func(a, b fileInfo) int { return cmp.Compare(filepath.Ext(a.Name()), filepath.Ext(b.Name())) }
	case "version":
		sorter = func(a, b fileInfo) int { return versCompare(a.Name(), b.Name()) }
	case "width":
		// TODO: maybe make it sort by display width (with quotes and all of
		// that)? That's what GNU ls does.
		sorter = func(a, b fileInfo) int { return cmp.Compare(len([]rune(a.Name())), len([]rune(b.Name()))) }
	case "none", "none-all":
		sorter, nameSort = nil, nil
	default:
		sorter, nameSort = nameSort, nil
	}
	if sorter != nil || sorter2 != nil {
		for _, p := range toPrint {
			if nameSort != nil {
				slices.SortFunc(p.fi, nameSort)
			}
			if sorter2 != nil {
				slices.SortStableFunc(p.fi, sorter2(p))
			} else {
				slices.SortStableFunc(p.fi, sorter)
			}
		}
	}
	if reverse {
		for _, p := range toPrint {
			slices.Reverse(p.fi)
		}
	}
	if dirsFirst {
		isdir := func(dir string, fi fileInfo) bool {
			if fi.IsDir() {
				return true
			}
			if fi.Mode()&fs.ModeSymlink == 0 {
				return false
			}
			// Symlink to dir should be counted as a "directory".
			l, err := os.Readlink(filepath.Join(dir, fi.Name()))
			if err != nil {
				return false
			}
			st, err := os.Stat(filepath.Join(dir, l))
			return err == nil && st.IsDir()
		}
		for _, p := range toPrint {
			sort.SliceStable(p.fi, func(i, j int) bool {
				return isdir(p.dir, p.fi[i]) && !isdir(p.dir, p.fi[j])
			})
		}
	}
	slices.SortFunc(toPrint, func(a, b printable) int {
		if a.isFiles {
			return -1
		}
		return cmp.Compare(a.dir, b.dir)
	})
}

// cmp(a, b) should return a negative number when a < b, a positive number when
// a > b and zero when a == b.
func versCompare(a, b string) int {
	if a == b {
		return 0
	}
	getNum := func(s string) (int, int, int) {
		var nonzero bool
		start, end, zeros := -1, -1, 0
		for i, c := range s {
			if start == -1 && isdigit(c) {
				if c == '0' {
					zeros++
				} else {
					nonzero = true
				}
				start = i
				continue
			}
			if start > -1 && c >= '1' {
				nonzero = true
			}
			if !nonzero {
				zeros++
			}
			if start > -1 && !isdigit(c) {
				end = i
				break
			}
		}
		if start > -1 && end == -1 {
			end = len(s)
		}
		return start, end, zeros
	}

	startA, endA, zeroA := getNum(a)
	if startA == -1 {
		return cmp.Compare(a, b)
	}
	startB, endB, zeroB := getNum(b)
	if startB == -1 {
		return cmp.Compare(a, b)
	}

	if zeroA != zeroB {
		return zeroB - zeroA
	}

	na, _ := strconv.ParseInt(a[startA:endA], 10, 64)
	nb, _ := strconv.ParseInt(b[startB:endB], 10, 64)

	return int(na - nb)
}

func isdigit(c rune) bool { return c >= '0' && c <= '9' }

type fakeFileInfo struct{ fs.DirEntry }

func (fakeFileInfo) ModTime() time.Time  { return time.Time{} }
func (fakeFileInfo) Sys() any            { return nil }
func (fakeFileInfo) Size() int64         { return -1 }
func (f fakeFileInfo) Mode() fs.FileMode { return f.Type() }
