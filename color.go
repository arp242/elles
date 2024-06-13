package main

import (
	"os"
	"runtime"
	"strings"

	"zgo.at/zli"
)

var (
	colorNormal, colorFile, colorDir, colorLink, colorPipe, colorSocket                 string
	colorBlockDev, colorCharDev, colorOrphan, colorExec                                 string
	colorDoor, colorSuid, colorSgid, colorSticky, colorOtherWrite, colorOtherWriteStick string
	reset                                                                               string

	defaultColors, defaultFromEnv = func() (string, bool) {
		if c := os.Getenv("ELLES_COLORS"); c != "" {
			return c, true
		}
		if c := os.Getenv("ELLES_COLOURS"); c != "" {
			return c, true
		}
		switch runtime.GOOS {
		case "freebsd", "openbsd", "netbsd", "dragonfly", "darwin":
			return "bsd", false
		default:
			return "gnu", false
		}
	}()
)

func setColor() {
	if !zli.WantColor {
		return
	}

	reset = zli.Reset.String()

	switch defaultColors {
	default:
		zli.Errorf("invalid ELLES_COLORS: %q", defaultColors)
	case "bsd":
		colorDir, colorLink = zli.Blue.String(), zli.Magenta.String()
		colorSocket, colorPipe = zli.Green.String(), zli.Yellow.String()
		colorExec, colorBlockDev = zli.Red.String(), (zli.Blue | zli.Cyan.Bg()).String()
		colorCharDev = (zli.Blue | zli.Yellow.Bg()).String()
		colorSuid = (zli.Black | zli.Red.Bg()).String()
		colorSgid = (zli.Black | zli.Cyan.Bg()).String()
		colorOtherWriteStick = (zli.Black | zli.Green.Bg()).String()
		colorOtherWrite = (zli.Black | zli.Blue.Bg()).String()
	case "gnu":
		colorDir, colorLink, colorPipe = "\x1b[01;34m", "\x1b[01;36m", "\x1b[33m"
		colorSocket, colorBlockDev, colorCharDev = "\x1b[01;35m", "\x1b[01;33m", "\x1b[01;33m"
		colorExec, colorDoor, colorSuid = "\x1b[01;32m", "\x1b[01;35m", "\x1b[37;41m"
		colorSgid, colorSticky, colorOtherWrite = "\x1b[30;43m", "\x1b[37;44m", "\x1b[34;42m"
		colorOtherWriteStick = "\x1b[30;42m"
	}
	if defaultFromEnv {
		return
	}
	if !readGNUColors() {
		readBSDColors()
	}
}

func readBSDColors() bool {
	c := os.Getenv("LSCOLORS")
	if c == "" {
		c = os.Getenv("LSCOLOURS")
		if c == "" {
			return false
		}
	}
	for i := range len(c) / 2 {
		var set *string
		switch i {
		case 0:
			set = &colorDir
		case 1:
			set = &colorLink
		case 2:
			set = &colorSocket
		case 3:
			set = &colorPipe
		case 4:
			set = &colorExec
		case 5:
			set = &colorBlockDev
		case 6:
			set = &colorCharDev
		case 7:
			set = &colorSuid
		case 8:
			set = &colorSgid
		case 9:
			set = &colorOtherWriteStick
		case 10:
			set = &colorOtherWrite
		default:
			// TODO: warn?
		}
		if col := (bsdcolor(c[i*2], false) | bsdcolor(c[i*2+1], true).Bg()); col == 0 {
			*set = ""
		} else {
			*set = col.String()
		}
	}
	return true
}

func bsdcolor(c byte, bold bool) zli.Color {
	if c >= 'a' && c <= 'h' {
		return zli.Black + zli.Color(c) - 0x61
	}
	if c >= 'A' && c <= 'H' {
		x := zli.Black + zli.Color(c) - 0x41
		if bold {
			x |= zli.Bold
		} else {
			x |= zli.Underline
		}
		return x
	}
	if c == 'X' {
		if bold {
			return zli.Bold
		}
		return zli.Underline
	}
	if c != 'x' {
		zli.Errorf("unknown color code in LSCOLORs: %c", c)
	}
	return 0
}

func readGNUColors() bool {
	c := os.Getenv("LS_COLORS")
	if c == "" {
		c = os.Getenv("LS_COLOURS")
		if c == "" {
			return false
		}
	}
	for _, cc := range strings.Split(c, ":") {
		if cc == "" {
			continue
		}
		k, v, ok := strings.Cut(cc, "=")
		if !ok {
			zli.Errorf("malformed LS_COLORS: %q", cc)
			continue
		}
		switch k {
		case "no":
			colorNormal = "\x1b[" + v + "m"
		case "fi":
			colorFile = "\x1b[" + v + "m"
		case "di":
			colorDir = "\x1b[" + v + "m"
		case "ln":
			colorLink = "\x1b[" + v + "m"
		case "pi":
			colorPipe = "\x1b[" + v + "m"
		case "so":
			colorSocket = "\x1b[" + v + "m"
		case "bd":
			colorBlockDev = "\x1b[" + v + "m"
		case "cd":
			colorCharDev = "\x1b[" + v + "m"
		case "or":
			colorOrphan = "\x1b[" + v + "m"
		case "ex":
			colorExec = "\x1b[" + v + "m"
		case "mi":
			// TODO: never applied; not entirely sure when it should get
			// applied, because as I read it, "mi" and "or" are both the same
			// thing: symlinks pointing to something that doesn't exist.
			//colorMissing = "\x1b[" + v + "m"
		case "do":
			colorDoor = "\x1b[" + v + "m"
		case "su":
			colorSuid = "\x1b[" + v + "m"
		case "sg":
			colorSgid = "\x1b[" + v + "m"
		case "st":
			colorSticky = "\x1b[" + v + "m"
		case "ow":
			colorOtherWrite = "\x1b[" + v + "m"
		case "tw":
			colorOtherWriteStick = "\x1b[" + v + "m"
		default:
			// Can't warn because of the "*.ext" stuff, which isn't implemented.
			//zli.Errorf("unknown key in LS_COLORS: %q", k)
		}
	}
	return true
}
