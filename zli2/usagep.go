package zli2

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"zgo.at/termtext"
	"zgo.at/zli"
)

// TODO: move to zli at some point.
//
// Many automatic usage generation tools are not very good. e.g. the Go flag
// package is horrible. Let's see if we can't do something better.
//
// The idea is to wrote usage messages "naturally" as plain readable text, and
// then we parse this.
//
// A header starts at column 1 and ends with a ":".
//
// Flags must always be indented with four spaces or a tab. The description can
// be over multiple lines as long as they start at the same or higher column:
//
// Header:
//
//     -flag1     Description.
//     --long     A flag with
//                more than one line
//                of description.
//     -a, -all   All!
//
// Text may be reflowed, depending on the output method. Use a blank line to
// start a new paragraph, or @@ at the end of the line to force a hard line
// break.
//
//     -flag1     Description.
//     --long     Must by one of:@@
//                   auto      Automatically determine it.@@
//                   never     Never do it.@@
//                   always    Always force it.@@
//     -a, -all   All!

type (
	Usage struct {
		flags    map[string]string
		Intro    string
		Sections []Section
	}
	Section struct {
		Title string
		Text  string
		Line  int
		Flags []Flag
	}
	Flag struct {
		Names []string
		Text  string
		Line  int
	}
)

var reFlag = regexp.MustCompile(`^(?:\t|    )(-(?:[a-zA-Z0-9_.=…-]+|,)(?:, )?)+`)

func Parse(s string) (Usage, error) {
	var (
		u       = Usage{flags: make(map[string]string)}
		lines   = strings.Split(strings.TrimSpace(s), "\n")
		curSect Section
		cur     strings.Builder
		skip    int
	)
	for i, l := range lines {
		if skip > 0 {
			skip--
			continue
		}
		l = strings.TrimRight(l, " \t")
		if l == "" {
			cur.WriteByte('\n')
			continue
		}

		if (l[0] != ' ' && l[0] != '\t') && strings.HasSuffix(l, ":") {
			if cur.Len() > 0 {
				if curSect.Title == "" {
					u.Intro = strings.TrimSpace(cur.String())
				} else {
					curSect.Text = strings.TrimSpace(cur.String())
					u.Sections = append(u.Sections, curSect)
				}
				cur.Reset()
				curSect = Section{Title: strings.TrimSuffix(l, ":"), Line: i + 1}
			}
			continue
		}

		m := reFlag.FindString(l)
		if m != "" {
			t := l[len(m):]
			fl := Flag{
				Line:  i + 1,
				Names: strings.Split(strings.TrimSpace(m), ", "),
				Text:  strings.TrimSpace(t),
			}
			off := len(m) + countSpace(t)
			for j := i + 1; j < len(lines); j++ {
				next := lines[j]
				if countSpace(next) < off {
					break
				}
				fl.Text += "\n" + strings.TrimLeft(next, " ")
				skip++
			}
			for _, n := range fl.Names {
				u.flags[strings.TrimLeft(n, "-")] = fl.Text
			}

			curSect.Flags = append(curSect.Flags, fl)
			continue
		}

		cur.WriteString(strings.TrimPrefix(l, "    "))
		cur.WriteByte('\n')
	}
	if cur.Len() > 0 {
		curSect.Text = strings.TrimSpace(cur.String())
		u.Sections = append(u.Sections, curSect)
	}
	return u, nil
}

func MustParse(s string) Usage {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func countSpace(s string) int {
	var n int
	for _, c := range s {
		if c != ' ' {
			break
		}
		n++
	}
	return n
}

// Section gets documentation for the section.
func (u Usage) Section(name string) (string, bool) {
	s, ok := u.flags[strings.TrimLeft(name, "-")]
	return s, ok
}

// flag gets documentation for the flag.
func (u Usage) Flag(name string) (string, bool) {
	s, ok := u.flags[strings.TrimLeft(name, "-")]
	return s, ok
}

// TODO: store flags map per section.
// func (s Section) Flag(name string) (string,bool _
// }

// Should print back the original.
func (u Usage) String() string {
	b := new(strings.Builder)
	b.WriteString(u.Intro)
	b.WriteByte('\n')

	for _, s := range u.Sections {
		fmt.Fprintf(b, "\n%s%s:%s\n", zli.Bold, s.Title, zli.Reset)
		if s.Text != "" {
			fmt.Fprintf(b, "\n    %s", strings.ReplaceAll(s.Text, "\n", "\n    "))
		}
		b.WriteByte('\n')
		for _, f := range s.Flags {
			// for i := range f.Names { f.Names[i] = zli.Colorize(f.Names[i], zli.Bold) }
			n := termtext.AlignLeft(strings.Join(f.Names, ", "), 16)

			//f.Text = termtext.WordWrap(strings.ReplaceAll(f.Text, "\n", " "), 58, strings.Repeat(" ", 22))
			f.Text = strings.ReplaceAll(f.Text, "\n", "\n"+strings.Repeat(" ", 22))
			fmt.Fprintf(b, "    %s  %s\n", n, f.Text)
		}
	}
	return b.String()
}

func (u Usage) Mandoc(name string, sect int) string {
	b := new(strings.Builder)

	fmt.Fprintf(b, ".Dd %s \n", time.Now().Format("January 2, 2006"))
	fmt.Fprintf(b, ".Dt %s %d\n", strings.ToUpper(name), sect)
	b.WriteString(".Os\n")
	b.WriteString(".Sh NAME\n")
	u.Intro = strings.TrimPrefix(u.Intro, name+" ")
	fmt.Fprintf(b, "%s – %s\n", name, u.Intro)

	for _, s := range u.Sections {
		fmt.Fprintf(b, ".Sh %s\n", strings.ToUpper(s.Title))
		if s.Text != "" {
			fmt.Fprintf(b, ".Pp\n%s\n", s.Text)
		}
		fmt.Fprintf(b, ".Bl -tag -width indent\n")
		// .It Fl D Ar format               → -D format
		// .It Fl -color Ns = Ns Ar when    → --color=when
		for _, f := range s.Flags {
			for i := range f.Names {
				f.Names[i] = strings.TrimPrefix(f.Names[i], "-")
			}
			fmt.Fprintf(b, ".It Fl %s\n", strings.Join(f.Names, " , Fl "))
			// TODO: marking flags like this breaks some stuff; need to look
			// into it
			// regexp.MustCompile(`-\w+`).ReplaceAllStringFunc(f.Text, func(s string) string {
			// 	s = strings.TrimLeft(s, "-")
			// 	if _, ok := u.Flag(s); ok {
			// 		return "\n.Fl " + s + "\n"
			// 	}
			// 	return s
			// })
			fmt.Fprintf(b, "%s\n", f.Text)
		}
		fmt.Fprintf(b, ".El\n")
	}
	return b.String()
}

// TODO: finish; to really get something decent we need at least:
//
// - A short description for flags.
// - Annotate which flags conflict.
// - Know if _files is correct.
// - Flags that can be doubled (-ll).
//
// And also:
//
// - Flags that take arguments, and what kind.
// - Positional arguments.
func (u Usage) CompleteZsh(name, site string) string {
	b := new(strings.Builder)
	b.WriteString(fmt.Sprintf(`#compdef %[1]s

# Completion for "elles"; %[2]s 
#
# Save as "_%[1]s" in any directory in $fpath; see the current list with:
#
#    print -l $fpath
#
# To add your own directory (before compinit):
#
#   fpath=(~/.zsh/funcs $fpath)

local arguments

arguments=(
`, name, site))

	flRepl := strings.NewReplacer(`'`, `\'`, "\n", " ")
	for _, s := range u.Sections {
		for _, f := range s.Flags {
			var n string
			if len(f.Names) > 1 {
				n = "{" + strings.Join(f.Names, ",") + "}"
			} else {
				n = f.Names[0]
			}
			fmt.Fprintf(b, "\t(%s)%s'[%s]'\n",
				strings.Join(f.Names, " "),
				n,
				flRepl.Replace(f.Text))
		}
		fmt.Fprintf(b, "\n")
	}

	b.WriteString("\t'*:file:_files'\n")
	b.WriteString(")\n\n_arguments -s -S : $arguments\n")
	return b.String()
}

// TODO
func (u Usage) CompleteBash() string {
	b := new(strings.Builder)
	return b.String()
}

// TODO
func (u Usage) CompleteFish() string {
	b := new(strings.Builder)
	return b.String()
}
