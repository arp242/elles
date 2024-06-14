package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"zgo.at/elles/os2"
	"zgo.at/zli"
)

const (
	_                  = 0
	borderToLeft uint8 = 1 << (iota - 1)
	alignNone
	alignLeft
)

type (
	col struct {
		s    string
		w    int
		prop uint8
	}
	cols struct {
		longest []int
		rows    [][]col
	}
	opts struct {
		list, quote, fullTime, maxColWidth int
		dirSlash, classify, comma          bool
		numericUID, group, hyperlink       bool
		blockSize, timeField               string
		one, cols, recurse, inode          bool
		noLinkTarget, trim                 bool
		octal                              bool
	}
)

func getCols(p printable, opt opts) cols {
	ncols := 1
	if opt.list == 1 {
		ncols = 3
	} else if opt.list >= 2 {
		ncols = 6
	}
	if opt.inode {
		ncols++
	}
	cc := cols{
		longest: make([]int, ncols),
		rows:    make([][]col, 0, len(p.fi)),
	}

	for i, fi := range p.fi {
		fp, afp := p.dir, p.absdir
		if p.isFiles {
			fp, afp = p.filePath[i], p.filePathAbs[i]
		}
		cur := make([]col, 0, ncols)
		if opt.list == 0 {
			if opt.inode {
				n := strconv.FormatUint(os2.Serial(p.absdir, fi), 10)
				cur = append(cur, col{s: n, w: len(n)})
			}

			n, w := decoratePath(fp, afp, fi, opt, false, !p.isFiles)
			cur = append(cur, col{s: n, w: w, prop: alignNone})
		} else if opt.list == 1 {
			s, w := listSize(fi, p.absdir, opt.blockSize, opt.comma)

			if opt.inode {
				n := strconv.FormatUint(os2.Serial(p.absdir, fi), 10)
				cur = append(cur, col{s: n, w: len(n)})
				cur = append(cur, col{s: s, w: w, prop: borderToLeft})
			} else {
				w++
				cur = append(cur, col{s: " " + s, w: w})
			}

			var t string
			switch opt.fullTime {
			case 0:
				t = shortTime(p.absdir, getTime(p.absdir, fi, opt.timeField))
			case 1:
				t = getTime(p.absdir, fi, opt.timeField).Format("2006-01-02 15:04:05")
			default:
				t = getTime(p.absdir, fi, opt.timeField).Format("2006-01-02 15:04:05.000000000 -07:00")
			}
			cur = append(cur, col{s: t, w: len(t), prop: borderToLeft})

			n, w := decoratePath(fp, afp, fi, opt, !opt.noLinkTarget, !p.isFiles)
			cur = append(cur, col{s: n, w: w, prop: borderToLeft | alignNone})
		} else {
			if opt.inode {
				n := strconv.FormatUint(os2.Serial(p.absdir, fi), 10)
				cur = append(cur, col{s: n, w: len(n)})
			}
			var perm string
			if opt.octal {
				m := fi.Mode() & 0o777
				if fi.Mode()&fs.ModeSticky != 0 {
					m |= 0o1000
				}
				if fi.Mode()&fs.ModeSetgid != 0 {
					m |= 0o2000
				}
				if fi.Mode()&fs.ModeSetuid != 0 {
					m |= 0o4000
				}
				perm = fmt.Sprintf("%4o", m)
			} else {
				perm = strmode(fi.Mode())
			}
			cur = append(cur, col{s: perm, w: len(perm)})

			user, group := owner(p.absdir, fi, opt.numericUID)
			cur = append(cur, col{s: user, w: len(user), prop: alignLeft})
			if opt.group {
				cur = append(cur, col{s: group, w: len(group), prop: alignLeft})
			} else if user != group {
				cur = append(cur, col{s: ":" + group, w: len(group) + 1, prop: alignLeft})
			} else {
				cur = append(cur, col{})
			}

			s, w := listSize(fi, p.absdir, opt.blockSize, opt.comma)
			cur = append(cur, col{s: s, w: w})

			var t string
			switch opt.fullTime {
			case 0:
				t = getTime(p.absdir, fi, opt.timeField).Format("Jan _2 15:04")
			case 1:
				t = getTime(p.absdir, fi, opt.timeField).Format("2006-01-02 15:04:05")
			default:
				t = getTime(p.absdir, fi, opt.timeField).Format("2006-01-02 15:04:05.000000000 -07:00")
			}
			cur = append(cur, col{s: t, w: len(t)})

			n, w := decoratePath(fp, afp, fi, opt, !opt.noLinkTarget, !p.isFiles)
			cur = append(cur, col{s: n, w: w, prop: borderToLeft | alignNone})
		}

		cc.rows = append(cc.rows, cur)
		var w int
		for i := range ncols {
			if cur[i].w > cc.longest[i] {
				cc.longest[i] = cur[i].w
			}
			w += cur[i].w
		}
	}
	return cc
}

func decoratePath(dir, absdir string, fi fs.FileInfo, opt opts, linkDest, listingDir bool) (string, int) {
	n := fi.Name()
	if dir != "" && !opt.recurse && !listingDir {
		n = filepath.Join(dir, n)
	}
	n = doQuote(n, opt.quote)

	// TODO: this should probably use zgo.at/termtext or something, pretty
	// sure alignment of this will be off in cases of double-width stuff
	// like some emojis, CJK, etc. termtext is relatively slow though;
	// should look into optimising the fast path on that.
	width := len([]rune(n))

	var didColor bool
	ifset := func(c string, class ...string) {
		if c != "" {
			didColor = true
			n = c + n + reset
		}
		if len(class) > 0 && (opt.classify || (class[0] == "/" && opt.dirSlash)) {
			n += class[0]
			width += len(class[0])
		}
	}
	ex := ""
	if fi.Mode()&0o111 != 0 {
		ex = "*"
	}
	switch {
	case fi.Mode()&fs.ModeSetuid != 0:
		ifset(colorSuid, ex)
	case fi.Mode()&fs.ModeSetgid != 0:
		ifset(colorSgid, ex)
	case fi.Mode().IsRegular():
		if ex != "" {
			ifset(colorExec, ex)
		} else {
			ifset(colorFile)
		}
	case fi.IsDir():
		switch {
		case fi.Mode()&0o002 != 0 && fi.Mode()&fs.ModeSticky != 0:
			ifset(colorOtherWriteStick, "/")
		case fi.Mode()&fs.ModeSticky != 0:
			ifset(colorSticky, "/")
		case fi.Mode()&0o002 != 0:
			ifset(colorOtherWrite, "/")
		default:
			ifset(colorDir, "/")
		}
	case fi.Mode()&fs.ModeNamedPipe != 0:
		ifset(colorPipe, "|")
	case fi.Mode()&fs.ModeSocket != 0:
		ifset(colorSocket, "=")
	case fi.Mode()&fs.ModeDevice != 0:
		ifset(colorBlockDev)
	case fi.Mode()&fs.ModeCharDevice != 0:
		ifset(colorCharDev)
	case os2.IsDoor(fi):
		ifset(colorDoor, ">")

	// Symlink
	case fi.Mode()&fs.ModeSymlink != 0:
		if !linkDest {
			ifset(colorLink, "@")
		} else {
			l, err := os.Readlink(filepath.Join(dir, fi.Name()))
			if err != nil {
				zli.Errorf(err)
			}

			fl := l
			if !filepath.IsAbs(fl) {
				fl = filepath.Join(dir, fl)
			}
			st, err := os.Stat(fl)
			var (
				c                = colorLink
				targetC, targetR string
			)
			if err != nil {
				if colorOrphan != "" {
					c, targetC, targetR = colorOrphan, colorOrphan, reset
				}
				if !errors.Is(err, os.ErrNotExist) && !os2.IsELOOP(err) {
					zli.Errorf(err)
				}
			} else if st.IsDir() {
				targetC, targetR = colorDir, reset
				if opt.classify {
					targetR += "/"
					width += 1
				}
			}

			l = doQuote(l, opt.quote)
			n = c + n + reset + " → " + targetC + l + targetR
			width += 3 + len(l)
		}
	}
	if !didColor {
		ifset(colorNormal)
	}

	if opt.hyperlink {
		hostnameOnce.Do(func() { h, _ := os.Hostname(); hostname = esc(h) })
		p := esc(filepath.Join(absdir, n))
		n = fmt.Sprintf("\x1b]8;;file://%s%s\a%s\x1b]8;;\a", hostname, p, n)
	}

	return filepath.ToSlash(n), width
}

var (
	hostname     string
	hostnameOnce sync.Once
)

// We don't want to escape slashes, but do want to replace everything else.
//
// TODO: do this properly; what we want is call that url.escape() function with
// encodePath (rather than encodePathSegment). That can only be done by
// constructing url.URL, setting Path, and calling EscapedPath(). Meh.
func esc(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), "%2F", "/")
}

func isVariationSelector(r rune) bool {
	return (r >= 0x180b && r <= 0x180f) || (r >= 0xfe00 && r <= 0xfe0f) || (r >= 0xe0100 && r <= 0xe01ef)
}

func doQuote(in string, level int) string {
	var (
		buf      = new(strings.Builder)
		dblQuote = level == 2
		n        = []rune(in)
	)
	buf.Grow(len(n))
	for i, r := range n {
		//if !unicode.IsPrint(r) || unicode.Is(unicode.Mn, r) {
		if !unicode.IsPrint(r) || isVariationSelector(r) {
			// Only display the brief escapes for \e, \n, \r, and \t as these
			// are fairly well-known. Who even knows what \v is?
			if level == 0 {
				buf.WriteString("$'")
			} else if level == 1 {
				dblQuote = true
			}
			switch r {
			case 0x1b:
				buf.WriteString(`\e`)
			case '\n':
				buf.WriteString(`\n`)
			case '\r':
				buf.WriteString(`\r`)
			case '\t':
				buf.WriteString(`\t`)
			default:
				if r < 0xff {
					fmt.Fprintf(buf, "\\x%02x", r)
				} else if r < 0xffff {
					fmt.Fprintf(buf, "\\u%04x", r)
				} else {
					fmt.Fprintf(buf, "\\U%08x", r)
				}
			}
			if level == 0 {
				buf.WriteString("'")
			}
		} else {
			// Always quote paths with leading and trailing spaces; hugely
			// confusing otherwise.
			if r == ' ' && (i == 0 || i == len(n)-1) {
				dblQuote = true
			}
			if level == 1 && needQuote(r) {
				dblQuote = true
			}
			if level >= 1 && (r == '"' || r == '`') {
				buf.WriteRune('\\')
			}
			buf.WriteRune(r)
		}
	}
	if dblQuote {
		buf.WriteByte('"')
		return `"` + buf.String()
	}
	return buf.String()
}

func needQuote(r rune) bool {
	switch r {
	case '|', '&', ';', '<', '>', '(', ')', '$', '\\', '"', '\'', ' ', // '
		'*', '?', '[', ']', '#', '~', '=', '%', '!', '`', '{', '}':
		return true
	}
	return false
}

func getTime(absdir string, fi fs.FileInfo, timeField string) time.Time {
	switch timeField {
	case "btime":
		return os2.Btime(absdir, fi)
	case "atime":
		return os2.Atime(fi)
	default:
		return fi.ModTime()
	}
}

func shortSize(n float64, u string) string {
	if n > 10 {
		return fmt.Sprintf("%.0f"+u, n)
	}
	return fmt.Sprintf("%.1f"+u, n)
}

func groupDigits(s string) string {
	i, frac, hasFrac := strings.Cut(s, ".")
	if hasFrac {
		frac = "." + frac
	}
	if strings.HasSuffix(s, "K") || strings.HasSuffix(s, "M") || strings.HasSuffix(s, "G") || strings.HasSuffix(s, "T") {
		frac += i[len(i)-1:]
		hasFrac = true
		i = i[:len(i)-1]
	}
	if len(i) <= 3 {
		return s
	}

	var (
		l = len(i) / 3
		r = len(i) % 3
		// +1 for dot, in case of frac. Not a big deal to over-alloc 1 byte.
		n = make([]byte, 0, len(i)+l+len(frac)+1)
	)
	if r != 0 {
		n = append(n, i[:r]...)
		n = append(n, ',')
	}
	for j := range l {
		j++
		if j > 1 {
			n = append(n, ',')
		}
		n = append(n, i[(j-1)*3+r:j*3+r]...)
	}
	if hasFrac {
		//n = append(n, '.')
		n = append(n, frac...)
	}
	return string(n)
}

func listSize(fi fs.FileInfo, absdir, blockSize string, comma bool) (string, int) {
	switch blockSize {
	case "s":
		s := strconv.FormatInt(os2.Blocks(fi), 10)
		if comma {
			s = groupDigits(s)
		}
		return s, len(s)
	case "S":
		bs := os2.Blocksize(filepath.Join(absdir, fi.Name()))
		s := strconv.FormatFloat(math.Ceil(float64(fi.Size())/float64(bs)), 'f', 0, 64)
		if comma {
			s = groupDigits(s)
		}
		return s, len(s)
	case "b", "B", "1":
		s := strconv.FormatInt(fi.Size(), 10)
		if comma {
			s = groupDigits(s)
		}
		return s, len(s)
	case "k", "K":
		s := shortSize(float64(fi.Size())/1024, "K")
		if comma {
			s = groupDigits(s)
		}
		return s, len(s)
	case "m", "M":
		s := shortSize(float64(fi.Size())/1024/1024, "M")
		if comma {
			s = groupDigits(s)
		}
		return s, len(s)
	case "g":
		s := shortSize(float64(fi.Size())/1024/1024/1024, "G")
		if comma {
			s = groupDigits(s)
		}
		return s, len(s)
	default:
		var s string
		if fi.Size() < 1024 {
			s = strconv.FormatInt(fi.Size(), 10)
		} else if fi.Size() < 1024*1024 {
			s = shortSize(float64(fi.Size())/1024, "K")
		} else if fi.Size() < 1024*1024*1024 {
			s = shortSize(float64(fi.Size())/1024/1024, "M")
		} else {
			s = shortSize(float64(fi.Size())/1024/1024/1024, "G")
		}
		if comma {
			s = groupDigits(s)
		}
		return s, len(s)
	}
}

// FileMode.String() doesn't align nicely with sticky bit and setuid. This ports
// strmode().
func strmode(m fs.FileMode) string {
	buf := make([]byte, 10)
	buf[0] = ftypelet(m)

	w := 1
	const rwx = "rwxrwxrwx"
	for i, c := range rwx {
		if m&(1<<uint(9-1-i)) != 0 {
			buf[w] = byte(c)
		} else {
			buf[w] = '-'
		}
		w++
	}
	if m&fs.ModeSetuid != 0 {
		buf[3] = 's'
		if m&0o100 == 0 {
			buf[3] = 'S'
		}
	}
	if m&fs.ModeSetgid != 0 {
		buf[6] = 's'
		if m&0o010 == 0 {
			buf[6] = 'S'
		}
	}
	if m&fs.ModeSticky != 0 {
		buf[9] = 't'
		if m&0o001 == 0 {
			buf[9] = 'T'
		}
	}
	return string(buf)
}

func ftypelet(m fs.FileMode) byte {
	const str = "d   lbps  c ?"
	for i, c := range str {
		if c != ' ' && m&(1<<uint(32-1-i)) != 0 {
			return byte(c)
		}
	}
	return '-'
	// Nonstandard file types.
	// if (S_ISCTG (m))
	//   return 'C';
	// if (S_ISDOOR (m))
	//   return 'D';
	// if (S_ISMPB (m) || S_ISMPC (m) || S_ISMPX (m))
	//   return 'm';
	// if (S_ISNWK (m))
	//   return 'n';
	// if (S_ISPORT (m))
	//   return 'P';
	// if (S_ISWHT (m))
	//   return 'w';
}

// Dislays as follows: just the time for today, "yst" and the time for
// yesterday, and "dby" and the time for the day before yesterday. Everything
// else displays as the date only "2006-01-02".
func shortTime(absdir string, tt time.Time) string {
	var (
		t   string
		now = time.Now()
	)
	if tt.Year() == now.Year() && tt.Month() == now.Month() && tt.Day() == now.Day() {
		t = tt.Format("15:04")
	} else if tt.Year() == now.Year() && tt.Month() == now.Month() && tt.Day() == now.Day()-1 {
		t = tt.Format("yst 15:04")
	} else if tt.Year() == now.Year() && tt.Month() == now.Month() && tt.Day() == now.Day()-2 {
		t = tt.Format("dby 15:04")
	} else {
		t = tt.Format("2006-01-02")
	}
	return t
}

// Cache this, as lookups are relatively expensive
var (
	users  []struct{ uid, n string }
	groups []struct{ gid, n string }
)

func owner(absdir string, fi fs.FileInfo, asID bool) (string, string) {
	uid, gid := os2.OwnerID(absdir, fi)
	if asID {
		return uid, gid
	}

	var uname, gname string
	for _, u := range users {
		if u.uid == uid {
			uname = u.n
			break
		}
	}
	for _, g := range groups {
		if g.gid == gid {
			gname = g.n
			break
		}
	}

	if uname == "" {
		u, err := user.LookupId(uid)
		if err != nil {
			u = &user.User{Username: uid}
			if uid == "" {
				u.Name = "[failed]"
			}
		}
		// On Windows "well-known sids" aren't mapped by user.LookupId. So copy
		// what "dir /Q" does.
		// https://learn.microsoft.com/en-us/windows/win32/secauthz/well-known-sids
		if runtime.GOOS == "windows" {
			switch uid {
			case "S-1-5-32-544":
				u.Username = `BUILTIN\Administrators`

			}
		}
		users = append(users, struct{ uid, n string }{uid, u.Username})
		uname = u.Username
	}

	if gname == "" {
		g, err := user.LookupGroupId(gid)
		if err != nil {
			g = &user.Group{Name: gid}
			if gid == "" {
				g.Name = "[failed]"
			}
		}
		groups = append(groups, struct{ gid, n string }{gid, g.Name})
		gname = g.Name
	}

	return uname, gname
}

func printJSON(toPrint []printable, errs *errGroup) {
	type (
		E struct {
			Name       string      `json:"name"`
			ModTime    time.Time   `json:"mod_time"`
			BirthTime  time.Time   `json:"birth_time"`
			AccessTime time.Time   `json:"access_time"`
			Type       fs.FileMode `json:"type"`
			Permission fs.FileMode `json:"permission"`
			Size       int64       `json:"size"`
		}
		J struct {
			Dir     string `json:"dir,omitempty"`
			Error   string `json:"error,omitempty"`
			AbsDir  string `json:"abs_dir,omitempty"`
			Entries []E    `json:"entries,omitempty"`
		}
	)
	var all []J
	for _, e := range errs.List() {
		all = append(all, J{Error: e.Error()})
	}
	for _, p := range toPrint {
		cur := J{Dir: p.dir, AbsDir: p.absdir, Entries: make([]E, 0, len(p.fi))}
		for _, fi := range p.fi {
			cur.Entries = append(cur.Entries, E{
				Name:       fi.Name(),
				ModTime:    fi.ModTime(),
				BirthTime:  os2.Btime(p.absdir, fi),
				AccessTime: os2.Atime(fi),
				Type:       fi.Mode().Type(),
				Permission: fi.Mode().Perm(),
				Size:       fi.Size(),
			})
		}
		all = append(all, cur)
	}

	out, err := json.MarshalIndent(all, "", "  ")
	zli.F(err)
	fmt.Fprintln(zli.Stdout, string(out))
	// err = jfmt.NewFormatter(min(columns, 80), "  ").Format(zli.Stdout, bytes.NewReader(out))
	// zli.F(err)
}
