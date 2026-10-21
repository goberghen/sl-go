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
└── sl-go_1.1.4-1_amd64.deb
```

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
sl --rule '\.tmp$ -> ignore' --rule '^IMG_[0-9]+\.JPG$ -> image' .
```

Option style is GNU: `--width 100` (or `--width=100`) for a long name, one dash
for a short one (`-a`, `-1`, `-o`, `-t`, `-w`, `-c`, `-l`, `-h`, `-V`, and
`-w100` for an attached value). The Tcl original also accepted long names behind
a single dash (`-width 100`); that is gone — one dash means a short option and
nothing else, and `sl` answers `-only` with `unknown option: -only (did you mean
--only?)` instead of guessing. `sl --help` prints everything.

### Notation

| mark | meaning |
|---|---|
| `/` | directory |
| `->` | symbolic link (`-->` when it points to another mount point) |
| `_` | special file (socket, FIFO, device) |
| `*` | executable |
| `0` | empty file or empty directory |
| `1` | directory holding exactly one entry |
| `<-` | more than one hard link |
| `X` | broken link, or an abandoned `~` backup |
| `u` `g` `s` | setuid, setgid, sticky |
| `W` | writable by everybody |
| `r` `x` | not readable / directory not searchable |
| `<2h` | changed recently (relative: among the newest here) |
| `-5m` | read recently |
| `4.2M` | relatively large |
| `...7` | seven consecutive names collapsed into this line |

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
