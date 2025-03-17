// vim:et:

package main

import (
	_ "embed"

	"zgo.at/elles/zli2"
)

//go:embed completion.zsh
var zsh string

var usage = zli2.MustParse(`
elles prints directory contents. https://github.com/arp242/elles

What to list:

    -a, -all         Show entries starting with . (except . and ..) or the
                     "hidden" attribute (on Windows)
    -d, -directory   List directories themselves, rather than their contents.
    -H               Follow symlinks of commandline arguments.
    -L               Follow all symlinks.
    -R, -recursive   List subdirectories recursively.
    -i, -inode       Print inode numbers.
    -g, -groupname   Always display the group by name in -ll; by default it's
                     only shown if the group group name is different from the
                     username.

How to list it:

    -j, -json        Print as JSON.
    -l               Long listing with size and mtime; use twice to show more.
    -1               List one path per line; default when stdout is not a tty
    -C               List paths in columns; default when stdout is a tty.
                     Single column (-1) is automatically set for -l, but can be
                     overridden with this.
    -group-dirs      Group directories first. Alias: -group-directories-first.
    -n               Display user an group ID as number, rather than username.
    -w, -width=..    Maximum column width; longer columns will be trimmed. Set
                     to 0 to disable.
    -m, -min=n       Minimum number of columns to use, trimming columns that are
                     too long. This does not set the exact number of columns and
                     sometimes results in more columns.
    -o, -octal       File permissions as octal instead of "rwx…".

How to format paths:

    -color=..        When to apply colours; always, never, or auto (default).
    -hyperlink=..    Add link escape codes; always, never (default), or auto.
    -p               Print / after each directory.
    -F               Print /@*=|> after directory, symlink, executable file,
                     socket, FIFO, or door.
    -,               (Comma) Print file sizes with thousands separators.
    -B, -blocks=..   Format for file sizes; as:
                       "1" or "B" for bytes
                       "s" for allocated filesystem blocks
                       "S" for blocks (differs from "s" for sparse files)
                       unit as K, M, or G (powers of 1024)
    -c               Use creation ("birth") time for display in -l, and sorting
                     with -t. Does nothing if neither -l nor -t is given.
    -u               Use last access time for display in -l, and sorting
                     with -t. Does nothing if neither -l nor -t is given.
    -T               Always display full time info, as "2006-01-02 15:00:00".
                     When given twice it will also display nanoseconds and
                     timezone.
    -Q               Quote paths with special shell characters or spaces; add
                     twice to always quote everything.
    -trim, -no-trim  Trim pathnames if they're too long to fit on the screen.
                     Only works for interactive terminals or when -w is set.
                     -no-trim turns this off and takes precedence over -trim (so
                     you can set -trim from an alias and turn it off).

Sorting:

    -r, -reverse     Reverse sort order.
    -S               By file size, largest first.
    -X               By file extension.
    -v               By natural numbers within text.
    -t               By modification time, newest first.
    -tc              By creation ("birth") time, newest first.
    -tu              By access time, newest first.
    -W               By pathname width (number of codepoints), shortest first.
    -f               Don't sort, list in directory order. Implies -a.
    -U               Don't sort, list in directory order.
    -sort=..         Sort by …: none (-U), size (-S), time (-t), version (-v),
                     extension (-X), width (-W)

Other:

    -help            Print this help and edit.
    -version         Print version and exit.
    -completion=..   Print shell completion file. Supported shells: "zsh".
    -manpage         Print manpage version of this help.

Environment:

    COLUMNS          Terminal width; falls back to ioctl if not set or 0.
    TZ               Timezone to use to for displaying dates.
    ELLES_COLORS     Colour configuration; see "Colours" section.
    LS_COLORS
    LSCOLORS

Colours:

    The defaults colours are identical to FreeBSD ls on all BSD systems and
    macOS, and GNU ls on everything else. The default colour scheme can be
    selected with ELLES_COLORS:

        ELLES_COLORS=bsd
        ELLES_COLORS=gnu

    The BSD defaults tend to work better on light backgrounds, and the GNU ones
    on dark backgrounds.

    Use LS_COLORS (GNU ls format) or LSCOLORS (BSD ls format) to configure the
    colours. It will try them in that order and use the first one that's found
    (on all platforms). Highlighting paths based on extension ("*.m4a=00;36")
    isn't implemented.

    ELLES_COLORS can accept additional :-separated values for elles-specific
    colouring features; this follos the GNU LS_COLORS syntax of «name»=«escape»,
    where «escape» is the terminal escape code to set.

        hidden  Additional highlights for hidden entries (e.g. those that start
                with a "."). These are applied after the regular colour codes.
                For example, to use the BSD defaults and add a grey background
                for hidden files:

                    ELLES_COLORS='bsd:hidden=48;5;255'

Compatibility flags:

    -G                Alias for -color=auto.
    -A, -almost-all   Alias for -a (both omit . and ..).
    -h                No-op, as elles uses human-readable sizes by default.
    -s                Alias for -blocks=s
`)
