This is mostly "just" `ls`, but a bit better. Nothing too fancy.

I wanted some flexibility in which columns to display and a few other minor
things; `ls` doesn't give you much options here. It started as a simple shell
script, and then became a simple Go program, and then things got rather of out
of hand.

Flags are sort-of compatible with `ls`, except when they're not. Full
compatibility with POSIX or any other `ls` isn't the main goal.

That said, most people should be able to use `alias ls=elles` and not get too
surprised; defaults are and the most commonly used flags are identical.

Installation
------------
There are binaries on the [releases] page; or to compile from source:

    go install -tags=osusergo zgo.at/elles@latest

Drop the `-tags=osusergo` to use libc for user lookups; only needed if you have
a complex setup with NIS or LDAP or whatnot. This will require a C compiler.

The Windows version is somewhat experimental.

[releases]: https://github.com/arp242/elles/releases

Usage
-----
The default is to display something rather similar to `ls`:

![`elles /`](ss/elles.png)

To get a more detailed listing, use the familiar `-l`:

![`elles -l /`](ss/elles_-l.png)

This displays a human-readable size, and the modification time as follows:

- just the time for today,
- "yst" and the time for yesterday,
- "dby" and the time for the day before yesterday, and
- everything else as the date only.

Add `-T` for a more complete date display:

![`elles -lT /`](ss/elles_-lT.png)

`-l` will print one entry per line by default, but you can combine that with
`-C`:

![`elles -lC /`](ss/elles_-lC.png)

This is the main reason I started working on this. `ls -l` is often too much
info. How often do I want to see the permission bits and number of links? Not
that often. How often does all of that chutney push the actual filenames off the
screen? Quite frequently.

For a while I used some scripts to modify the `ls` output, which works well
enough for the common case, but also not really for rather a lot of uncommon
cases. Many of these uncommon cases are quite common.

Anyway, use `-l` twice for more details:

![`elles -ll /`](ss/elles_-ll.png)

This is more similar to the standard `ls -l` output, with some small
differences. `-C` also works for this:

![`elles -llC /`](ss/elles_-llC.png)

Sometimes things are annoying to display because of long filenames; for example
listing my `~/.cache` doesn't even fit on a single window:

![`elles ~/.cache`](ss/elles_.cache.png)

This one 79-character `event-sound-cache..` file forces single-column display.
With the `-w` option you can set the maximum column width to deal with this sort
of nonsense:

![`elles -w30 ~/.cache`](ss/elles_-w_30_.cache.png)

There's a bunch of other useful flags. See `elles -help` for, well, help.

Differences from POSIX
----------------------
There are some intentional differences from POSIX 2017. This started as a small
list of a few items, but has rather grown.

- `-c` uses create ("birth") time, rather than ctime (which is usually identical
  to mtime, and is rarely useful for display or sorting).

- `-u` and `-c` behave simpler: show update/create time when given, and sort
  with that when `-t` is also given. This is so much simpler than the whole
  "sort by last access time, unless `-l` is given, in which case it will be used
  for display and NOT sorting, unless `-t` is also given, in which case it will
  be used for display and sorting".

- `-l` output is much shorter; `-l -l` (or `-ll`) is more similar to POSIX `-l`,
  but without the number of links (doesn't seem useful to me).

- The `-l` or `-ll` output won't print a `total: …` line for directories. I
  don't think I've ever used it. Use `du` for this.

- `-g` and `-o` for `-l` without group or owner are not implemented, as it's
  somewhat pointless since `-l` doesn't print either by default.

- `-a` works like `-A`; I don't see why you ever want to include `.` and `..`;
  seems like backwards compatibility with 1971 Unix.

- `-L` (dereference ALL symlinks) is not implemented; don't see why you'd ever
  want to follow *all* symlinks. Instead, `-L` prevents showing the symlink
  targets in `-l`.

- `-m` for CSV-y "Stream output format" is not implemented. Doesn't seem too
  useful and also error-prone (doesn't escape `,`). Use shell globs or `-json`.

- `-x` (sort across) is not implemented. Never used it. Send patch if you want
  it.

- `-k` to set the blocksize to 1024 is not implemented as POSIX blocksize
  semantics are stupid.

- `-s` uses the filesystem's block size rather than 512 or 1024 bytes. POSIX
  blocksize semantics are stupid.

- It will print with "human-readable" file sizes by default (`-h` on most `ls`
  implementations, but not in POSIX). Use `-B`/`-block` to set a different
  blocksize.

- `-q` is not implemented; I don't see when this would ever be useful.

- When sorting by time (`-t`), files with the same time are sorted ascending,
  like everything else, rather than descending. This inconsistency in sorting is
  a weird POSIX quirk that exists for $reasons.

TODO
----
- Sorting is a simple byte sort (as LC_COLLATE=C); the golang.org/x/text/collate
  package is based on Unicode 6.2, from 2012, and generally seems fairly
  unmaintained, with a number of reported bugs. I'll have to find or write an
  alternative I guess...

  Also want it to be configurable; I would like en.UTF-8 sorts *AND* sorting
  capitals before lower case like with C, so this can be used to "pin" paths on
  top. I was never able to get that to work with ls (so I just use LC_COLLATE=C)
  and x/text/collate also doesn't support it.

- There is no way to display file flags, ACLs, MAC labels, whiteouts,
  capabilities, or anything like that.

- Look into displaying sparse files better. A 8G sparse file will show up as 8G,
  even though it has 0 allocated blocks. You can use `-s`, but should be obvious
  from the standard output.

- There isn't really any way to customize the time format other than `-T` and
  `-TT`. Just a fixed `+[..]` like most other tools doesn't seem quite right,
  because I rather like the "display only time for today, something else for
  other days" type logic. Also might want to do relative times ("5 hours ago")
  as: "relative for this week, full date for older".

- Can't configure which borders to display, or column width (FreeBSD ls has
  LS_COLWIDTHS for that).

- I didn't implement any filtering; not sure yet what the best approach for
  this. Realistically, I almost never want to see `*.o` files in my listing. eza
  has `--git-ignore`, which seems to have a huge potential for confusion: it's
  pretty common to have compiled binaries in there too, or cache directories, or
  other things you really want in your listing. Overall, seems more of a footgun
  than helpful.

  GNU and eza both have -I/--ignore, but don't really want to construct custom
  paths either. GNU has -B/--ignore-backups to ignore files ending with `~`.
  Maybe a "ignore common files you almost never want" option might be the best,
  and a (non-intrusive) hint that we're ignoring *some* files.

- Display "git status". I don't really want to tie elles to git; maybe something
  like:

      % elles -ext='git status --porcelain'

  That will call the `-ext` tool for every argv, in this case just the current
  directory. That will output:

       M README.md
       M main.go
       M main_test.go
       M print.go
       M usage.go
      ?? new.go

  And then take everything before the first space as status, and everything
  after that as the pathname (which we can use to match it up).

  Or something along those lines. This can then be used with other VCS tools, or
  other clever stuff.
