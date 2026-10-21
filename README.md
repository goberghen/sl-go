# sl — a more useful `ls`

`sl` takes the most common use of `ls` — showing a directory compactly in
columns — and makes it *informative*. It groups files by purpose, marks the ones
that stand out, condenses runs of numbered names, and lays the result out to fit
your terminal.

```text
$ sl
  build           doc                                                video
  Makefile <90d   empty.txt0                                         episode 01.mkv...6
  README           long file name that will have/be sho...here.txt   other
  dev             image                                              notes.md
  main.c          cover.png                                          archive
  src/2           audio                                              archive.tar.gz
                   song.mp3 195K
  15 files (210K) + 1 .dotfile + 1 ignored, 1 directory (2 files)
```

Read that as: two source files in `dev` plus a subdirectory of two; an empty text
file; a long name shortened in the middle (keeping the head and the tail); a
`video` block where six episodes collapsed into one line; and an archive.
The bottom line is the summary.

This is a Go port of the Tcl script **sl — usefully organize directories and
point out interesting files** by Thomas A. Phelps (v1.1.3, revision 1.89,
February 2012, GPLv3). The original home page,
<http://practicalthought.com/sl/>, is offline since about 2017 and is preserved at
<https://web.archive.org/web/2016/http://practicalthought.com/sl/>.

## What it does

* **Groups by purpose**, in a fixed order: `dir`, `build`, `app`, `dev`,
  `A2` (Apple II disk images), `WWW`, `aux`, `doc`, `book`, `document`,
  `image`, `vector`, `audio`, `video`, `font`, `data`, `other`, `archive`,
  `dotfile`, `Mac`, `ignore`.
* **Classifies directories by content**: a folder of `.mp3` is shown as audio, a
  folder with `index.html` as a web root, a `VIDEO_TS` folder as video.
* **Marks what is interesting**, relative to the rest of the directory: the
  biggest files, the recently changed, the recently read, empty files, empty
  directories, directories holding a single entry, unreadable entries,
  world-writable directories, hard links, broken symlinks, odd permission bits,
  and runs of names that form a series.
* **Condenses**: `frame 001.dng … frame 342.dng` becomes `frame 001.dng...342`,
  and a shared prefix can be elided (`--prefix 3`) so a column of
  `myproject-notes` / `myproject-todo` shows only the differing tails.
* **Filters noise** by default: object files, `.pyc`, C headers, `CVS`, `TAGS`,
  editor backups. `-a` shows everything, `--ignore N` picks the level.
* **Lays out for reading**: per-column widths, group titles, middle-ellipsis for
  long names, and a summary line.

`sl` is not a replacement for `ls` in scripts: the output is coloured, adaptive
and full of heuristics. It is a tool for a human looking at a directory.

## Install

```sh
make                      # release/bin/sl-<os>-<arch> + a source tarball
sudo make install         # copies it to /usr/local/bin/sl, plus the man page
make deb                  # or: release/sl-go_1.1.4-1_amd64.deb (/usr/bin/sl inside)
```

Every binary is named after the platform it was built for — `sl-linux-amd64`,
not a bare `sl` — because a bare name says nothing about what is inside. The
installed name is always `sl`; renaming is what `make install` and the packaging
recipes do, not something you have to do by hand.

Everything a build produces goes to `sl-go/release/`, which is empty in a fresh
checkout, listed in `.gitignore`, and removed by `make clean`. Nothing is ever
written above the source directory (`make OUT=/some/dir` if you want it elsewhere):

```text
sl-go/release/
├── bin/sl-<os>-<arch>  one renamed file per platform — the host build after
│      `make`, all twelve after `make all`:
│      sl-linux-{amd64,arm64,386,arm,ppc64,ppc64le}, sl-darwin-{amd64,arm64},
│      sl-freebsd-amd64, sl-openbsd-{amd64,arm64}, sl-netbsd-amd64
├── sl-go-1.1.4.tar.gz          sources, refreshed by every `make`
├── sl-binaries-1.1.4.tar.gz    all twelve binaries in one archive (`make bundle`)
└── sl-go_1.1.4-1_amd64.deb
```

To hand the binaries to somebody — a USB stick, a container image, a colleague on
another platform — use the archive rather than the twelve loose files:

```sh
make bundle              # release/sl-binaries-1.1.4.tar.gz
tar tzf release/sl-binaries-1.1.4.tar.gz | head -3
#   sl-binaries-1.1.4/README.txt
#   sl-binaries-1.1.4/SHA256SUMS
#   sl-binaries-1.1.4/bin/sl-darwin-amd64
```

It carries the man page and the completions too, is reproducible, and its
`SHA256SUMS` is relative to the archive root, so `sha256sum -c SHA256SUMS` works
straight after unpacking. Details in `INSTALL.md` §3.10.

Every binary is pure Go (`CGO_ENABLED=0`) and — except on OpenBSD, where the Go
toolchain always links the base system's libc — static: the Linux ones run on
glibc and musl alike, so there is no separate "musl build". `make check-binaries`
verifies that, and with `ALPINE_ROOTFS=...` it also runs the binary inside
Alpine. The BSD builds need nothing on the building machine; `packaging/freebsd/`
and `packaging/openbsd/` carry the port skeletons, `packaging/nix/` + `flake.nix`
and `packaging/guix/` carry the Nix and Guix recipes. i386, ARM and the two
PowerPC flavours build from the same source; 32-bit PowerPC does not exist as a
Go target.

The tarball is deterministic: sorted names, fixed owner and timestamp, so the same
sources always produce the same archive (compare `md5sum`), and it is built from a
`sl-go-<version>/` top-level directory that unpacks and builds with a plain
`make`.

The Debian package is called **`sl-go`** (this is the Go port) and installs
`/usr/bin/sl` and `sl(1)`. It declares `Conflicts: sl`, so it cannot be
installed next to Debian's own `sl` (the steam locomotive); install one or the
other, and the package manager will ask before swapping them.

Shell completions for bash, zsh and fish are installed by every package; they are
generated from the option table (`make completions`) and a test keeps them in
step with `--help`. On Debian/Ubuntu the zsh file goes to
`/usr/share/zsh/vendor-completions/_sl`, elsewhere to
`/usr/share/zsh/site-functions/_sl` — Debian's zsh does not read the latter, and
a file in the wrong directory does nothing at all. `sl -<TAB>` then lists the
options, `sl --<TAB>` the long ones, each with the same sentence `--help` gives
for it.

Full instructions, including the Debian package, cross-compilation and
uninstallation: **[INSTALL.md](INSTALL.md)**. Manual page: `man sl` or
[`docs/sl.1`](docs/sl.1).

## Usage

```sh
sl                        # the current directory
sl ~/Downloads
sl -a /usr/include        # include the normally hidden headers
sl --only /usr/bin        # only entries that carry some signal
sl --title --nosfx --prefix 3 ~/Music
sl --color=never -1 .     # auto (default), always, never
sl --sort size ~/Downloads   # biggest first, sizes shown (implies --relsize 100)
sl --sort time --one .       # most recently modified first
sl --rule '\.tmp$ -> ignore' --rule '^IMG_[0-9]+\.JPG$ -> image' .
```

Option style is GNU: `--width 100` (or `--width=100`) for a long name, one dash
for a short one (`-a`, `-1`, `-o`, `-t`, `-w`, `-c`, `-l`, `-h`, `-V`, and
`-w100` for an attached value). The Tcl original also accepted long names behind
a single dash (`-width 100`); that is gone — one dash means a short option and
nothing else, and `sl` answers `-only` with `unknown option: -only (did you mean
--only?)` instead of guessing. `sl --help` prints everything.

### Notation

Marks sit right after the name, in a fixed order, and the type marks come first:

```text
  Makefile <-W<90d     hard link, world-writable, changed 90 days ago
  empty.txt0           empty file
  src/2                a directory holding two files
  broken-link->X       a symbolic link that goes nowhere
  4.2M                 relatively large — the size *is* the mark
  chapter1.txt...6     six names collapsed into one line (see --series)
```

| mark | meaning | what to do about it |
|---|---|---|
| `/` | directory | — |
| `->` | symbolic link (`-->` when it points to another mount point) | — |
| `_` | special file (socket, FIFO, device) | — |
| `*` | executable | — |
| `0` | empty file or empty directory | a leftover, usually safe to delete |
| `1` | directory holding exactly one entry | a wrapper: `--inline-singleton` shows `src/main.c` instead |
| `<-` | more than one hard link | deleting this name does not free the space |
| `X` | broken link, or an abandoned `~` backup | dead weight |
| `u` `g` `s` | setuid, setgid, sticky | intentional? if not, look closer |
| `W` | writable by everybody | a permission mistake in most cases |
| `r` `x` | not readable / directory not searchable | you will not get far into it |
| `<2h` | changed recently (relative: among the newest here) | what you were just working on |
| `-5m` | read recently | what you just looked at |
| `4.2M` | relatively large | the top `--relsize` percent of this directory |
| `...7` | seven consecutive names collapsed into this line | `--series 0` to see them all |

A listing is a grid, and a grid fills **downwards**: the first column is read top
to bottom, then the second, and so on. That is the order the entries are in, so
pull a listing into one column with `-1` when the order is the point.

The three groups answer different questions: **type** (what it is), **anomaly**
(something to fix), **relative** (how it compares to its neighbours — a
percentage of *this* directory, so it changes as the directory does).

Two consequences worth knowing:

* a marked entry is **not** collapsed into a series. Eight freshly created
  `episode 0N.mkv` all carry `<NOW`, so they stay eight lines; give them an old
  timestamp and they become one. This is the Tcl original's rule, kept on
  purpose — the mark is the reason the file was worth pointing out.
* when names are cut to fit, they are first *rewritten*: articles are dropped and
  prepositions become `/`, so `The Making of a Documentary about the Sea volume
  001 of the series.txt` is shown as `Making/Documentary about Sea/001/series.txt`.
  `--shorten 0` shows names exactly as they are on disk.

## Examples

Every block below is real output from the commands shown, in the directory this
snippet builds:

```sh
mkdir demo && cd demo && mkdir season1
printf 'int main(void){return 0;}\n' > main.c
printf 'int util(void){return 1;}\n' > util.c
touch main.o
head -c 40000 /dev/urandom > cover.png
for i in 1 2 3; do head -c 900000 /dev/urandom > "track 0$i.mp3"; done
for i in 1 2 3 4 5 6; do echo x > "chapter$i.txt"; done
for i in 1 2 3 4 5 6 7 8; do head -c 250000 /dev/urandom > "season1/episode 0$i.mkv"; done
echo '# readme' > README.md; echo 'all:' > Makefile
: > empty.txt; ln -s missing-target broken-link; mkdir world; chmod 777 world
echo x > .hidden
touch -d 2020-01-01 chapter*.txt
```

### What is in here at all

```text
$ sl --width 100
  world/0W <NOW    doc                 audio                      other
  build            chapter1.txt...6     track 01.mp3 879K <NOW    README.md <NOW
  Makefile <NOW    empty.txt0 <NOW     track 02.mp3 <NOW          broken-link->X <NOW
  dev              image               track 03.mp3 <NOW
  main.c <NOW      cover.png <NOW      video
  util.c <NOW                          season1/8 <NOW
  15 files (2.6M) + 1 .dotfile + 1 ignored, 2 directories (8 files)
```

Two source files in `dev`, six chapters collapsed into one line, three tracks in
`audio`, the episode folder named `season1` with a count of eight, an empty file,
a broken link, a world-writable directory — and the ignored `.o` plus the
dotfile counted in the summary instead of being shown.

### The things that are usually hidden

```text
$ sl -a --width 100 | tail -4
  dev              empty.txt0 <NOW     track 03.mp3 <NOW          dotfile
  main.c <NOW      image               video                      .hidden <NOW
  main.o0 <NOW     cover.png <NOW      season1/8 <NOW
  17 files (2.6M), 2 directories (8 files)

$ sl --ignore 2 --width 100 | tail -3
  main.c <NOW      image               video
  main.o0 <NOW     cover.png <NOW      season1/8 <NOW
  16 files (2.6M) + 1 .dotfile, 2 directories (8 files)
```

Level 3 (default) hides both, `--ignore 2` shows the object file, level 1 also
the dotfile, `--ignore 0` is `-a`.

### Only what stands out

`-o` drops every entry without a mark — the boring majority:

```text
$ sl -o --width 100
  world/0W <NOW    util.c <NOW        audio                video
  build            doc                track 01.mp3 <NOW    season1/8 <NOW
  Makefile <NOW    empty.txt0 <NOW    track 02.mp3 <NOW    other
  dev              image              track 03.mp3 <NOW    README.md <NOW
  main.c <NOW      cover.png <NOW
```

Note what goes with it: `-o` implies `--relsize 0 --series 0 --prefix 0
--summary 0` and turns `--title` off, so the tracks keep their names, the
chapters come back one by one, no sizes are shown and there is no summary line.
Options given *after* `-o` win.

### Series, and why they sometimes do not collapse

Freshly created episodes all carry `<NOW`, and a marked run is left alone:

```text
$ sl season1
  video                        episode 03.mkv <NOW    episode 06.mkv <NOW
   episode 01.mkv 244K <NOW    episode 04.mkv <NOW    episode 07.mkv <NOW
  episode 02.mkv <NOW          episode 05.mkv <NOW    episode 08.mkv <NOW
  8 files (1.9M)
```

Give them an old timestamp and the same eight names become one line:

```text
$ touch -d 2020-01-01 season1/*
$ sl season1
  video
   episode 01.mkv 244K <2020...8
  8 files (1.9M)
```

`--series 0` shows every name; `--series 3` lowers the threshold to three.

### Shared prefixes

```text
$ sl --one --prefix 4          # myproject-notes.txt, -plan.txt, -todo.txt, other.txt
  dev
   myproject-notes.txt 5B <NOW
  doc
  myproject-plan.txt <NOW
           -todo.txt <NOW
  other.txt <NOW
  4 files (14B)

$ sl --one --prefix 0
  dev
   myproject-notes.txt 5B <NOW
  doc
  myproject-plan.txt <NOW
  myproject-todo.txt <NOW
  other.txt <NOW
```

The prefix becomes spaces, so the column still lines up, and it only applies
inside a group and only against the entry just above. Be aware it can elide more
than you expect — digits are part of the prefix too:

```text
$ sl --one --prefix 4          # track 01.mp3, track 02.mp3
  audio
   track 01.mp3 879K <NOW
        02.mp3 <NOW
  2 files (1.7M)
```

### Cutting long names

Twelve names, each `The Making of a Documentary about the Sea volume 0NN of the
series.txt`:

```text
$ sl --width 120
  doc                                                          Making/Documentary about Sea/007/series.txt <NOW
    Making/Documentary about Sea/001/series.txt 1B <NOW        Making/Documentary about Sea/008/series.txt <NOW
   Making/Documentary about Sea/002/series.txt <NOW            Making/Documentary about Sea/009/series.txt <NOW
   Making/Documentary about Sea/003/series.txt <NOW            Making/Documentary about Sea/010/series.txt <NOW
   Making/Documentary about Sea/004/series.txt <NOW            Making/Documentary about Sea/011/series.txt <NOW
   Making/Documentary about Sea/005/series.txt <NOW            Making/Documentary about Sea/012/series.txt <NOW
  The Making of a Documentary about the Sea volume 006 of the series.txt <NOW
  12 files (12B)
```

Eleven of the twelve got shorter and one did not — shortening stops as soon as
the line fits. Cutting is three steps, cheapest first: drop articles, turn
prepositions into `/`, and only then elide the middle (`--ellipsis` changes that
last step, and it needs at least two columns to do anything). That is why the
shown name can differ from the real one — `--shorten 0` shows them as they are on
disk:

```text
$ sl --width 120 --shorten 0 | head -3
  doc
   The Making of a Documentary about the Sea volume 001 of the series.txt 1B <NOW
  The Making of a Documentary about the Sea volume 002 of the series.txt <NOW
```

### A single-child chain

`--expando` is **not** recursion. It unfolds `project/project-1.0/src` style
chains, where each level holds nothing but the way down:

```text
$ sl --expando 0 project
  build          dev
  README <NOW     src/3 <NOW
  1 file (2B), 1 directory (3 files)

$ sl project
  build          dev
  README <NOW     src/3 <NOW
  1 file (2B), 1 directory (3 files)

autoexpand src/:
  deep/1 <NOW    dev                util.c <NOW
                  main.c 2B <NOW
  2 files (4B), 1 directory (1 file)

autoexpand deep/:
  dev
   a.c 2B <NOW
```

As soon as a level holds a second subdirectory or more than `--expando N` files,
it stops. There is no depth flag and no whole-tree walk.

### Your own marks

This is the column `eza` has for git — but for any command. The hook runs in the
directory being listed, with the path as `$1` and the name as `$2`, and only its
first line is used (cut to 24 characters):

```text
$ sl --annotate 'git status --porcelain -- "$1" | head -1'
  doc                        tracked.txtM tracked.txt 8B <24s
  new.txt?? new.txt <24s
  2 files (12B) + 1 .dotfile
```

One process per entry is fine for a handful of names. For a directory, use
`--annotate-batch`: it gets every path on stdin and must answer one line per
path:

```text
$ sl --annotate-batch 'while IFS= read -r p; do
      if [ -f "$p" ]; then echo "$(stat -c%s "$p")b"; else echo dir; fi
  done'
  world/0W dir <19s    empty.txt0 0b <19s
  build                image
  Makefile 5b <19s     cover.png 40000b <19s
  dev                  audio
  main.c 26b <19s       track 01.mp3 900000b 879K <19s
  util.c 26b <19s      track 02.mp3 900000b <19s
```

### Sorting

Inside each group the default order is dictionary order by name. `--sort` changes
that; four notes and three images, with sizes and timestamps deliberately
different:

```text
$ sl --one --sort size
   aaa/1 <NOW
   zzz/5 <NOW
  image
   c.png 586K
-> a.png 195K -2y
   b.png 29K
  other
   draft.md 293K
   readme.md 88K
   notes.md 4.9K
   todo.md 1B
  7 files (1.2M), 2 directories (6 files)
```

Every size is printed although `--relsize` was not given: `--sort size` implies
`--relsize 100`, otherwise only the largest few entries would carry a number and
the order would look random. An explicit `--relsize` wins, and so does `-o`.

`--relsize 100` means exactly that — every value, smallest included. Below 100
the cutoff is clamped to the median so that a directory where everything is
nearly the same size does not light up as "relatively large".

The two directories stay in name order even though `zzz` holds five entries and
`aaa` one: a directory's size here is the *number of entries in it*, not the
number of bytes below it, so sorting by it would claim that a folder of small
files outweighs a large video. Use `--sort time` for those.

```text
$ sl --one --sort time
  aaa/1 <NOW
  zzz/5 <NOW
  image
   c.png 586K
  b.png
->a.png -2y
  other
  todo.md
  readme.md
  notes.md
  draft.md
  7 files (1.2M), 2 directories (6 files)

$ sl --one --sort atime
  aaa/1 <NOW
  zzz/5 <NOW
  image
->a.png -2y
   c.png 586K
  b.png
  other
  notes.md
  todo.md
  readme.md
  draft.md
  7 files (1.2M), 2 directories (6 files)
```


`time` is the modification time (what `--relchange` marks), `atime` the access
time (what `--relread` marks) — two different orders, as the columns show.

In a grid the same order reads **down the column**, not across the line:

```text
$ sl --width 100 --sort size
   aaa/1 <NOW    image        -> a.png 195K -2y    other              readme.md 88K     todo.md 1B
   zzz/5 <NOW     c.png 586K     b.png 29K          draft.md 293K     notes.md 4.9K
  7 files (1.2M), 2 directories (6 files)
```

First column: `aaa`, `zzz`; second: `image`, `c.png 586K`; third: `a.png`,
`b.png`; fourth: `other`, `draft.md 293K`; fifth: `readme.md 88K`, `notes.md`;
sixth: `todo.md` — the sizes fall all the way through, 586K, 195K, 29K, 293K, 88K,
4.9K, 1B. Use `-1` when the order matters more than the shape.

Two things `--sort` does **not** do: it never moves an entry to another group,
and it switches `--prefix` off (a shared prefix is elided against the entry above,
and after a sort that neighbour is somebody else). Series collapsing runs first,
on name order, and a collapsed run keeps the place it was given.

### Other things worth knowing

```sh
sl --rule '\.bak$ -> archive' --rule '(?i)^(notes|todo)'   # force groups
sl --autosearch '(?i)key|password' ~/Downloads             # highlight matches
sl --relchange 50 --relsize 20 --width 100                 # mark more than usual
sl --sort size --relsize 20 .                              # sort, but mark only the top fifth
sl --no-config --k1000                                     # ignore the config, decimal units
sl --title --nosfx                                         # prettify names, drop suffixes
```

`--help` lists every option; `man sl` has the same examples with more detail.

## Configuration

The first run **creates** `$XDG_CONFIG_HOME/sl/config` (usually
`~/.config/sl/config`): a plain `key = value` file in which every setting is
present, commented out, with its default value next to it. So the file changes
nothing until you uncomment a line, and it doubles as the reference documentation
for the settings. An existing file is never overwritten; delete it to get a fresh
copy at the next start. `--no-config` skips both reading and creating it.

The format is flat `key = value` (plus `style.<name>`, `group.<name>.extra`,
`annotate*`), deliberately not TOML: the program depends on nothing but the Go
standard library, and what the file describes — a handful of scalars and two
dotted namespaces — does not need a format with tables, arrays of tables and
dates. See INSTALL.md §5 for the reasoning.

```ini
# what to show
ignore = 2            # also show C headers (.h) and objects (.o)
relread = 0           # this file system does not update atime
# what to condense
series = 3
prefix = 3
# extra suffixes for a group
group.doc.extra = .md .rst
```

```ini
ignore = 2                  # also show .h and .o
relread = 0                 # this file system keeps no useful atime
prefix = 3
style.dir = 01;34
group.doc.extra = .md .rst
annotate = git log -1 --format=%cr -- "$1"
annotate-match = \.c$
```

### User rules

The original could be extended by redefining procedures in `~/.sl.tcl`. A static
binary cannot host an interpreter, so the same job is done declaratively: a rule
forces a group on every name that matches a regular expression.

```ini
rule = \.tmp$ -> ignore           # hide build leftovers
rule = \.(md|rst)$ -> doc         # treat Markdown as documentation
rule = ^IMG_[0-9]+\.JPG$ -> image # the camera's uppercase extension
rule = (?i)^makefile$ -> build
```

The pattern is Go syntax, unanchored and case-sensitive unless it starts with
`(?i)`; it is matched against the entry's name with the compression wrapper
stripped (`foo.1.gz` is matched as `foo.1`). Rules are tried in order, the first
match wins, and a rule overrides the built-in tables. `ignore` hides the entry,
like a built-in ignorable name (visible again with `--all`). `--rule` takes the
same text on the command line and wins over the file. An unknown group or a
broken pattern stops the program at startup. `man sl` has the full description
(section RULES) and the list of groups.

Every long option works as a key. `--annotate` replaces the original's Tcl hooks
(`perfile`, `vc-*`): for each matching entry a command is run and its first line
of output is appended to the entry — that is how you get a git age, a line count
or a build number next to a file.

`--annotate-batch` does the same with **one process per directory** instead of
one per entry: the paths arrive on the command's standard input, one per line,
and it answers with one line per path.

```sh
# 1200 files: 1535 ms as --annotate, 28.5 ms as --annotate-batch
sl --annotate 'echo "$2"' .
sl --annotate-batch 'while read -r p; do git log -1 --format=%cr -- "$p"; done' .
```

The command becomes a filter (the directory is in `$1`), and a name containing a
newline cannot be sent (such entries stay unannotated). If the command answers
with a different number of lines than it got, the answers would land on the wrong
names, so sl says so once and shows no annotations for that directory. The two
hooks are mutually exclusive; `man sl` has the details.

## Differences from the Tcl original

The grouping tables, warning marks, relative thresholds, series collapsing,
prefix elision and column heuristics are reproduced from the original. What
differs:

* no embedded Tcl: `~/.sl.tcl` is not executed, `perfile`/`vc-*` become
  `--annotate`, configuration is data in `~/.config/sl/config`;
* deterministic order (entries are sorted, not left in `readdir` order), which
  can change *which* of several tied entries is picked as the group's
  representative;
* colours honour `NO_COLOR` and `LS_COLORS`; `--color` takes `auto`, `always`
  or `never` (`--no-color` is the same as `--color=never`);
* user rules (`rule = <regexp> -> <group>`, `--rule`) replace what `~/.sl.tcl`
  was used for;
* Unicode-aware column widths (CJK, combining marks, emoji);
* the Apple II group also knows post-2012 image formats (`.woz`, `.hdv`, `.d13`,
  `.edsk`, `.edd`, `.nb2`, `.n35`, `.d16`) and the 13-sector `.nib` size;
* the contents of a directory are classified by their **real type** (`d_type`)
  instead of by "no suffix, so probably a subdirectory", so `README`,
  `Makefile` and `LICENSE` stop being counted as folders; `--legacy-classify`
  restores the old rule;
* a **name series** needs one run of digits (or a single index letter at the end
  of the stem), so `episode01`/`episode02` and `part a`/`part b` collapse but
  `abc.txt`/`abd.txt` no longer do; `--legacy-series` restores the old rule;
* `--owner-notes` (on by default) names the owner of files in your home
  directory that are not yours — root leftovers, another account, a container
  bind mount; `--group-mix` shows `5+3` instead of `8` on a directory classified
  by its contents, i.e. how confident the guess was;
* `--experimental` (the original's TAP branch, now an alias for
  `--group-mix --owner-notes`) and `--inline-singleton` (the branch the original
  left disabled: `src/main.c` instead of `src/1`);
* `--sort` orders entries inside a group by size, modification time or access
  time. The original always printed name order, and grouping, series collapsing
  and prefix elision are kept as they were (see `--sort` in the manual);
* not ported: RCS/CVS marks, the Apple resource-fork size fallback, and the
  `touch`-based access time hack.

## Development

```sh
make test     # unit tests, golden listing, Unicode widths, Apple II group
make vet      # go vet + gofmt check
make all      # linux/darwin × amd64/arm64
make dist     # release/sl-go-<version>.tar.gz
make deb      # Debian package (needs dpkg-deb)
make packages # recipes filled in with the release checksum -> release/packaging/,
              # and one SHA256SUMS for every artifact -> release/SHA256SUMS
make bench    # in-process benchmarks, matcher vs regexp
make bench-ab # wall clock of two binaries, interleaved: A=old/sl B=release/bin/sl-linux-amd64
make check-binaries # verify each binary's linkage (static on Linux/BSD except OpenBSD, libc on OpenBSD/macOS)
make verify   # make all, then check-binaries
make completions # regenerate completions/{sl.bash,_sl,sl.fish} from the option table
make layout   # where sources, docs and build output live
```

Layout of the tree:

```text
sl-go/
├── Makefile              build, test, install, package
├── README.md  INSTALL.md  LICENSE
├── docs/sl.1             manual page (roff)
├── completions/          bash, zsh and fish completion files (generated)
├── tools/completions/    the generator for the three above
├── packaging/            build-deb.sh, sl-go.spec, PKGBUILD, APKBUILD,
│                         sl-go.rb, freebsd/, make-packages.sh
├── bench/                bench.go (wall clock, A/B) and results/
├── src/                  all Go sources
│   ├── main.go cli.go config.go initconfig.go hooks.go lscolors.go width.go
│   ├── names.go scan.go series.go relative.go entry.go pipeline.go display.go layout.go
│   ├── stat_unix.go stat_linux.go stat_bsd.go
│   └── *_test.go         the tests
├── release/              build output, empty by default, wiped by `make clean`
└── go.mod
```

Tests sit next to the code, as `_test.go` files, because that is the only place Go
allows them for a `package main`: a separate `tests/` directory can only import a
library package, and `main` cannot be imported. They are still separate files and
`make test` runs them all.

The code follows the original's pipeline — collect → mark up → order → lay out →
print — so the Tcl script remains a useful reference when reading it.

## License

GPL-3.0-or-later, as the original script. No warranty. See [LICENSE](LICENSE).
Original: Copyright (c) 2011–2012 Thomas A. Phelps.
Go port: 2026, goberghen & Arena.ai agents.

The port continues the original's version line: the Tcl script reached v1.1.3, so
this is v1.1.4.
