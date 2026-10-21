# Installing sl

`sl` is a single binary: no runtime, no interpreter, no dependencies beyond
a Unix-like system. It writes its configuration file by itself on the first run.

The program is the Go port of Thomas A. Phelps' Tcl script *sl — usefully
organize directories and point out interesting files* (v1.1.3, revision 1.89,
February 2012, GPLv3). The port itself is v1.1.4, by goberghen & Arena.ai agents.
The original home page, <http://practicalthought.com/sl/>, has been offline since
about 2017; it is preserved at
<https://web.archive.org/web/2016/http://practicalthought.com/sl/>.

---

## 1. Requirements

| | |
|---|---|
| to build | Go 1.21 or newer (`go version`), `make` (optional but convenient) |
| to run | Linux (glibc **and** musl), macOS, FreeBSD, NetBSD, OpenBSD (the OpenBSD build links the base system's libc, see §3.6) |
| to build the `.deb` | `dpkg-deb` (from `dpkg-dev`), `gzip`, `tar` |

The Linux, FreeBSD, NetBSD and macOS binaries are statically linked and about
2 MB. They have no external dependencies: the whole program uses the Go standard
library only, so the same Linux file runs on a glibc distribution and on musl
(Alpine, distroless, `FROM scratch`) without a second build. `make check-binaries`
proves it, see §3.5.

OpenBSD and macOS are the exceptions, and they are not ours to change: on both,
Go's runtime makes its system calls **through the system C library** — libc on
OpenBSD, libSystem on macOS — because neither OS offers a stable ABI for raw
system calls. The Go source says it in as many words
(`runtime/sys_openbsd_amd64.s`: "System calls are implemented in
libc/libpthread, this file contains trampolines"), the binary really does import
64 libc functions (`access`, `kevent`, `kqueue`, `__thrsleep`, …), and
`-linkmode internal` changes nothing: the calls are still there. A truly static
OpenBSD binary would mean patching the Go runtime to issue raw system calls,
i.e. maintaining a toolchain fork against an OS that deliberately does not
guarantee them — out of scope, and the dynamic binary needs nothing installed
anyway: `libc.so` and `libpthread.so` are part of every OpenBSD base system.

So the rule is: **static where the platform allows it** (Linux, FreeBSD, NetBSD,
where the runtime issues raw calls), **libc-linked where it does not** (OpenBSD,
macOS). `make check-binaries` enforces exactly that rule per platform — it is not
"a Linux check": FreeBSD and NetBSD binaries are held to the same static
requirement as the Linux ones, and if the OpenBSD or macOS linkage ever changes
for another reason, the check says so.

Per-OS support is not a promise, it is a build tag: `stat_linux.go`,
`stat_bsd.go` plus `st_time_bsd.go`/`st_time_openbsd.go` (OpenBSD names the
timestamp fields `Mtim`/`Atim`, everyone else `Mtimespec`/`Atimespec`), and
`make all` builds linux, darwin, freebsd, openbsd and netbsd in one go.

---

## 2. Build from source

```sh
git clone <this repository> sl-go        # or unpack the tarball
cd sl-go
make                 # -> release/bin/sl-linux-amd64 and release/sl-go-1.1.4.tar.gz
```

`make` builds for the machine you are on and names the file after it (here a
Linux/amd64 host):

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" \
    -o release/bin/sl-linux-amd64 ./src
```

Nothing is built under the bare name `sl`: the file would be a byte-for-byte
copy of one of the platform binaries, and a name that does not say which OS and
architecture it was built for is worse than no name at all. `make install` (and
every package recipe) installs it into `$PATH` as `sl`.
Nothing is written into the source tree: binaries and packages go to
`release/` inside the source directory; `make OUT=/some/dir ...` puts it elsewhere.

Cross-compile every target at once:

```sh
make all             # release/bin/sl-{linux,darwin,freebsd,openbsd,netbsd}-<arch>
```

Or pick your own:

```sh
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o release/bin/sl-freebsd-amd64 ./src
```

The tree:

```text
sl-go/
├── Makefile              build / test / install / package
├── README.md  INSTALL.md  LICENSE
├── docs/sl.1             manual page
├── packaging/            deb, rpm, PKGBUILD, APKBUILD, Homebrew, FreeBSD
├── bench/                wall-clock A/B harness and recorded results
├── src/                  all sources and their _test.go files
└── go.mod
```

The tests live in `src/` next to the code, because Go only allows `_test.go` files
in the package's own directory, and `package main` cannot be imported from a
separate `tests/` directory.

Run the tests:

```sh
make test            # unit tests, golden listing, Unicode width, Apple II group
```

One test set is opt-in because it needs a populated filesystem. The byte-level
matchers in `src/match.go` replace hot regular expressions, and the test proves
each of them agrees with the regexp it replaces on real names:

```sh
go test ./src -run OnCorpus -corpus /usr,/etc,/opt -v
```

It walks the given trees, compares every name against the original pattern and
stops at the first disagreement. On a normal Linux install that is roughly
130 000 names and takes about two seconds.

Measure instead of guessing:

```sh
make bench                                        # in-process; matcher vs regexp
make bench-ab A=old/sl B=release/bin/sl-linux-amd64 DIRS="/usr/bin /etc"
```

`bench/README.md` explains what each number does and does not include; the runs
quoted in the release notes are kept in `bench/results/`.

---

## 3. Install

### 3.1 By hand

```sh
sudo install -m 0755 release/bin/sl-linux-amd64 /usr/local/bin/sl
sudo install -D -m 0644 docs/sl.1 /usr/local/share/man/man1/sl.1
sudo gzip -9 /usr/local/share/man/man1/sl.1
```

or, the same thing, `sudo make install`.

Without root:

```sh
install -D -m 0755 release/bin/sl-linux-amd64 ~/.local/bin/sl
install -D -m 0644 docs/sl.1 ~/.local/share/man/man1/sl.1
```

Make sure `~/.local/bin` is in `PATH`.

### 3.2 Debian/Ubuntu package

The Debian package is called **`sl-go`** (this is the Go port) and it installs
the usual names:

```text
/usr/bin/sl                        the binary
/usr/share/man/man1/sl.1.gz        the manual page
/usr/share/doc/sl-go/              README, INSTALL, copyright, changelog
```

It also declares

```text
Conflicts: sl
Replaces:  sl
Provides:  sl
```

because Debian has its own `sl` package (the steam locomotive that runs across
the terminal), which wants the same `/usr/bin/sl` and `sl(1)`. The two packages
are therefore **mutually exclusive**: apt refuses to install both, and when you
install ours over the locomotive it asks before removing it. That is the
intended behaviour — one `sl` on the machine, and it is this one.

```sh
make deb             # -> release/sl-go_1.1.4-1_amd64.deb
sudo apt install ./release/sl-go_1.1.4-1_amd64.deb
#   or: sudo dpkg -i release/sl-go_1.1.4-1_amd64.deb
```

If the locomotive is installed, `apt install` will tell you that `sl` has to be
removed first (`Conflicts: sl`) and will do it as part of the same transaction.
If you would rather keep it, `apt remove sl-go` first, or use the manual
`make install` into `/usr/local` described above — `/usr/local/bin` comes before
`/usr/bin` in `PATH`, so the two can coexist there.

Building for another architecture:

```sh
ARCH=arm64 make deb                  # uses release/bin/sl-linux-arm64, builds sl-go_..._arm64.deb
ARCH=all   make deb                  # every binary in one package
```

Inspect before installing:

```sh
dpkg-deb -I release/sl-go_1.1.4-1_amd64.deb     # metadata
dpkg-deb -c release/sl-go_1.1.4-1_amd64.deb     # file list
man sl                                          # after installation
```

Uninstall:

```sh
sudo apt remove sl-go
```

`make deb` is a one-line wrapper around `packaging/build-deb.sh`; run the script
directly if `make` is not installed, or to pass more variables
(`REVISION=2 MAINTAINER="..." OUT=/tmp/pkg ./packaging/build-deb.sh`).

### 3.3 RPM (Fedora, RHEL, openSUSE)

`packaging/sl-go.spec` builds the same file set from the source tarball:

```sh
make dist                        # release/sl-go-1.1.4.tar.gz
mkdir -p ~/rpmbuild/{SPECS,SOURCES,BUILD,RPMS}
cp release/sl-go-1.1.4.tar.gz ~/rpmbuild/SOURCES/
rpmbuild -ba packaging/sl-go.spec      # fill @VERSION@ and @DATE@ first
```

The spec declares `Conflicts: sl`, `Obsoletes: sl < 5`, `Provides: sl`, so
`dnf install sl-go` replaces the locomotive rather than failing on the file
conflict.

### 3.4 Arch Linux, Alpine, Homebrew, FreeBSD

The recipes are short (one static binary, one man page), so all of them live in
`packaging/`:

| file | system |
| --- | --- |
| `packaging/PKGBUILD` | Arch Linux / AUR |
| `packaging/APKBUILD` | Alpine Linux |
| `packaging/sl-go.rb` | Homebrew (Intel and Apple Silicon) |
| `packaging/freebsd/` | FreeBSD port skeleton (`sysutils/sl-go`) |
| `packaging/openbsd/` | OpenBSD port skeleton (`sysutils/sl-go`) |
| `packaging/nix/default.nix` + `flake.nix` | Nix / NixOS |
| `packaging/guix/sl-go.scm` | Guix |

Each one needs the release tarball and its checksum, which is what
`make packages` is for — it refreshes the tarball, computes the checksum, fills
the placeholders in every recipe and stages the result:

```sh
make packages
# -> release/packaging/{PKGBUILD,APKBUILD,sl-go.rb,sl-go.spec,sl-go.scm,nix-default.nix,flake.nix,freebsd-*,openbsd-*}
#    and one checksum file for the whole release: release/SHA256SUMS

cd release/packaging && makepkg -si          # Arch
abuild -r                                    # Alpine
brew install --build-from-source ./sl-go.rb  # Homebrew
```

Every recipe installs `/usr/bin/sl` and `sl(1)` and declares the conflict with
the other `sl`, for the reason given above; each one builds with
`CGO_ENABLED=0`, so the result is a static executable with no runtime
dependencies — including on Alpine, where that is unusual.

The tarball is byte-for-byte reproducible (sorted names, fixed owner and
timestamp), so the checksums in these files can be verified by rebuilding
`sources` from the same tree, and `make packages` can be re-run without
invalidating a submitted recipe.

### 3.5 musl, Alpine, distroless: the same binary

There is nothing to "build for musl". `CGO_ENABLED=0` means the binary contains
no libc code at all — no glibc, no musl — and makes the syscalls itself. So
`release/bin/sl-linux-amd64` (and `sl-linux-arm64`) is the right file for Debian,
for Alpine, for a `FROM scratch` image and for an initramfs.

Prove it rather than take it on trust:

```sh
make all             # 12 binaries: static on Linux/FreeBSD/NetBSD,
                     # dynamic against the system libc on OpenBSD and macOS (see below)
make check-binaries  # verify each of them against what its platform requires
make verify          # make all + check-binaries

# and, if you have an Alpine root filesystem around:
curl -LO https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-minirootfs-3.20.3-x86_64.tar.gz
mkdir -p /tmp/alpine && sudo tar xzf alpine-minirootfs-*.tar.gz -C /tmp/alpine
sudo ALPINE_ROOTFS=/tmp/alpine make check-binaries
#   check-binaries: release/bin/sl-linux-amd64: static, no NEEDED entries
#   check-binaries: release/bin/sl-linux-amd64: runs on musl: sl 1.1.4
```

There is no "static build" to ask for: `CGO_ENABLED=0` is the only build mode, so
`make all` has no static variant. The targets differ only in what they do:

| target | builds | verifies |
| --- | --- | --- |
| `make all` | 12 binaries | — |
| `make check-binaries` | — (needs `make all` first) | every binary, against the linkage its platform requires |
| `make verify` | 12 binaries | the same check |

`packaging/check-binaries.sh` is the script behind the target. It reads the ELF
`OS/ABI` field, so it knows which rule to apply:

* Linux, FreeBSD, NetBSD — **must be static**: no `NEEDED` entries, no `.interp`;
* OpenBSD — **is dynamic by design** (Go's port calls libc; see §3.6): exactly
  `libc.so` and `libpthread.so` are expected, and anything else fails the check;
* macOS — Mach-O with `libSystem`, which Apple requires.

It also skips architecture-mismatched binaries instead of trying to run them (an
arm64 file will not execute in an x86_64 chroot, and saying so is more useful
than a segfault).

In a container image the whole installation is two lines:

```dockerfile
FROM scratch
COPY sl-linux-amd64 /sl
ENTRYPOINT ["/sl"]
```

On Alpine the source package is the normal route (`packaging/APKBUILD`, §3.4),
and it produces the same static file.

### 3.6 OpenBSD (and NetBSD)

```sh
make all                                     # release/bin/sl-openbsd-amd64, sl-openbsd-arm64
CGO_ENABLED=0 GOOS=openbsd GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o sl ./src
```

The Go toolchain on Linux cross-compiles these; nothing of OpenBSD is needed to
build them, only to run them.

The OpenBSD binaries are **dynamically linked** — `libc.so` and `libpthread.so`,
both part of the base system:

```text
$ readelf -d release/bin/sl-openbsd-amd64 | grep NEEDED
 (NEEDED)  Shared library: [libc.so]
 (NEEDED)  Shared library: [libpthread.so]
```

`CGO_ENABLED=0` removes cgo, but on OpenBSD the Go runtime still calls libc for
system calls, so the linker emits a normal dynamic executable. Nothing has to be
installed to run it (the two libraries ship with the OS), but it is not a static
binary and should not be described as one. The FreeBSD and NetBSD builds have no
`NEEDED` entries at all. A port skeleton, mirroring the FreeBSD one, is in
`packaging/openbsd/` (Makefile, `pkg/DESCR`, `pkg/PLIST`, conflicts with the old
`sl-ls` package that carried the Tcl original).

One real bug came out of doing this: the tree advertised `openbsd` in the build
tag of `stat_bsd.go` but did not compile there, because OpenBSD's `struct stat`
calls the timestamp fields `Mtim`/`Atim` while FreeBSD, NetBSD and Darwin call
them `Mtimespec`/`Atimespec`. The fields are now behind `stMtime`/`stAtime` in
`st_time_bsd.go` and `st_time_openbsd.go`, and `make all` builds all five
systems.

### 3.7 Nix and Guix

**Nix.** The repository is a flake (`flake.nix` in the root) and the derivation
is `packaging/nix/default.nix`:

```sh
nix build .#sl-go            # -> result/bin/sl
nix run .#sl-go -- --version
nix profile install .#sl-go
nix-build packaging/nix      # the same without flakes
```

The program has no dependencies, so the derivation has no `vendorHash` to
maintain (it is `null`, and stays correct as long as `go.mod` has no requires).
`CGO_ENABLED=0` is `buildGoModule`'s default, which is what this program wants,
so the store path works on musl-based systems too. To use it as an overlay:
`nixpkgs.overlays = [ sl.overlays.default ];` then `pkgs.sl-go`.

**Guix.** `packaging/guix/sl-go.scm` is a package definition for
`go-build-system`:

```sh
guix install -L packaging/guix sl-go
```

Two details that are easy to get wrong and are therefore commented in the file:
the import path is `sl` (that is what `go.mod` says, not the project name), and
the unpacked directory is `sl-go-<version>`, hence `#:unpack-path`. Guix
checksums are written in **base32**, not hex; `make packages` fills that in
(`@BASE32@` → 52 characters), so the submitted recipe does not have to be
finished by hand.

Both recipes were written against the tree that `make all` and `make test`
verify, but **neither nix nor guix is installed in the environment this port was
developed in**, so the recipes themselves are syntax-reviewed, not evaluated.
If one of them fails, the failure will be in the recipe, not in the program: the
Go build they wrap is exactly `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"`
from §2.

### 3.8 Other architectures: i386, PowerPC, ARM, RISC-V

`make all` builds twelve targets; the exotic ones are built the same way, one
command each:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=386     go build -trimpath -ldflags "-s -w" -o sl-386 ./src
CGO_ENABLED=0 GOOS=linux GOARCH=ppc64   go build ... -o sl-ppc64 ./src
CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le go build ... -o sl-ppc64le ./src
GOARM=5 CGO_ENABLED=0 GOOS=linux GOARCH=arm go build ...    # ARMv5, otherwise GOARM=7
```

| target | built here | run here | notes |
| --- | --- | --- | --- |
| `linux/amd64`, `linux/arm64` | yes | yes | the usual pair |
| `linux/386` | yes | **yes** (ia32 emulation) | output identical to amd64 on 198 comparison cells |
| `linux/arm` (GOARM=7) | yes | no | Raspberry Pi and friends; `GOARM=5` for ARMv5 |
| `linux/ppc64` | yes | no | big-endian PowerPC (POWER8+, ELFv1) |
| `linux/ppc64le` | yes | no | little-endian PowerPC (ELFv2) |
| `linux/riscv64`, `linux/s390x`, `linux/loong64` | yes | no | compile cleanly, not shipped |
| `freebsd/386`, `openbsd/386`, `netbsd/386` | yes | no | 32-bit BSDs, same fix as i386 |
| **`linux/ppc`** (32-bit) | — | — | **impossible**: Go has no `ppc` architecture at all |

The last row is the answer to "and 32-bit PowerPC?": the Go toolchain rejects
the pair outright (`go: unsupported GOOS/GOARCH pair linux/ppc`). There is no
`GOARCH=ppc` in any Go release; the 64-bit `ppc64`/`ppc64le` are the only
PowerPC targets. A 32-bit PPC build would mean porting to C or gccgo, i.e. a
different program.

Endianness is a non-issue here by construction: sl reads directory entries and
metadata through the operating system and never parses a binary format of its
own, so there is no byte-order-dependent code. That is a statement about the
code, not a test result: the ppc64 binaries could not be executed in this
environment (no qemu-ppc64), only built and inspected with `file`.

### 3.9 Shell completions

The three completion files are ordinary files in the tree — `completions/sl.bash`,
`completions/_sl` (zsh), `completions/sl.fish`. There is no `--completion` flag
and the program prints nothing on demand: completion data belongs to the
package, and a `sl` in the middle of a TAB press would be a process start on
every key.

They are **generated from the option table** in `src/cli.go`, so a new option
cannot be forgotten:

```sh
make completions      # rewrites the three files; the result is committed
make install          # also installs them to /usr/local/share/...
```

Generated once, then checked: `TestCompletionsMatchOptions` fails if any option,
short flag, `--color` value or group name from `--rule` is missing from any of
the three files, and `TestCompletionsKnowEveryGroup` compares the group list with
the program's own `groupNames()`. That is the failure mode worth preventing — a
completion file that offers an option that no longer exists.

Every package recipe installs them where each shell looks:

```text
/usr/share/bash-completion/completions/sl        bash (loaded automatically)
/usr/share/zsh/vendor-completions/_sl            zsh, Debian/Ubuntu
/usr/share/zsh/site-functions/_sl                zsh, everyone else
/usr/local/share/zsh/site-functions/_sl          zsh, `make install`
/usr/share/fish/vendor_completions.d/sl.fish     fish
```

Where each recipe puts `_sl`, and why the directory really is in `$fpath` there:

| recipe | installed path | in `$fpath` because |
| --- | --- | --- |
| `.deb` | `/usr/share/zsh/vendor-completions` | Debian's zsh sets its site-fndir to that directory |
| `sl-go.spec` (Fedora) | `/usr/share/zsh/site-functions` | the spec sets `sitefndir=%{_datadir}/zsh/site-functions` when building zsh |
| `PKGBUILD` (Arch) | `/usr/share/zsh/site-functions` | Arch's zsh only overrides `--enable-fndir`, the site-fndir stays the default |
| `APKBUILD` (Alpine) | `/usr/share/zsh/site-functions` | same: only `--enable-fndir` is overridden |
| FreeBSD port | `/usr/local/share/zsh/site-functions` | the zsh port builds with `--disable-site-fndir --enable-additional-fpath=${DATADIR}/site-functions` |
| OpenBSD port | `/usr/local/share/zsh/site-functions` | zsh's default site-fndir with `PREFIX=/usr/local`, which is also zsh's built-in default entry |
| `make install` | `/usr/local/share/zsh/site-functions` | `/usr/local/share/zsh/site-functions` is the first entry of zsh's built-in `fpath` |
| Nix, Guix | `$out/share/zsh/site-functions` | only if the user's setup adds that directory to `fpath` — see below |
| Homebrew | `<prefix>/share/zsh/site-functions` | works out of the box on Intel (`/usr/local`), needs an `fpath` line on Apple Silicon (`/opt/homebrew`) |

So: the same directory for everything except Debian, which is the only packaging
in the list that moves its site-fndir elsewhere.

**The zsh directory is not the same everywhere, and getting it wrong fails
silently.** Debian's zsh does *not* have `/usr/share/zsh/site-functions` in
`$fpath` — the directory exists, but the packaged location is
`/usr/share/zsh/vendor-completions`. A `_sl` in the wrong place is simply never
read, and `sl -<TAB>` falls back to filenames, exactly as if no completion file
existed. That was a real bug in the `.deb`; it now installs to
`vendor-completions`, and both the path (`TestDebZshCompletionPath`) and the
behaviour (`TestZshCompletionListsOptions`, which starts an interactive zsh in a
pty and presses TAB) are tested.

Nix and Guix are the two cases a recipe cannot fix by itself: nixpkgs builds zsh
with `--disable-site-fndir`, and both store packages in per-package directories.
On NixOS, `programs.zsh.enableCompletion = true` (or home-manager's zsh module)
puts the system's `share/zsh/site-functions` on `fpath`, and the completion shows
up; with a bare `nix profile install` you have to add it yourself:

```sh
fpath=(~/.nix-profile/share/zsh/site-functions $fpath)
```

If completion does not work after installing:

```sh
print -l $fpath | grep zsh        # is the directory where _sl lives in $fpath?
ls /usr/share/zsh/*/ _sl          # where is _sl actually?
compinit -u; rm -f ~/.zcompdump   # rebuild the completion dump, open a new shell
```

What is completed: all long options with their help text (zsh and fish show it
next to the match — `--width -- assume a terminal N columns wide`, the way
`dpkg`'s completion does), the short flags, the three values of `--color`, the
four ignore levels, and the group names after `--rule RE ->`. Values that are
paths (`--config`, `--startup`, `--mounts`) are left to the shell's own file
completion, `--annotate`/`--annotate-batch` offer command names, and options that
take a number or a regex offer nothing rather than a list of filenames.

The descriptions are the same strings `--help` prints, read out of the option
table by the generator, so they cannot drift: `TestCompletionsCarryDescriptions`
fails if any option's help is missing from `_sl` or `sl.fish`.

**A real bug came out of trying i386.** `stat_linux.go` read
`time.Unix(st.Mtim.Sec, st.Mtim.Nsec)`, which does not compile on a 32-bit
target, because `Timespec.Sec` is `int32` there and `time.Unix` wants `int64`:

```text
src/stat_linux.go:18:23: cannot use st.Mtim.Sec (variable of type int32) as int64 value
```

So "i386 support" was, like OpenBSD, a claim nobody had ever built. The
conversion is now explicit (`int64(st.Mtim.Sec)`) in `stat_linux.go` and in
`stat_bsd.go`, which also unblocked `freebsd/386`, `openbsd/386` and
`netbsd/386`.

Build artifacts live in `release/` inside the source tree, and nowhere else:

```text
release/bin/sl-linux-amd64        the host build (all twelve after `make all`)
release/sl-go-1.1.4.tar.gz       sources, refreshed by every `make` / `make dist`
release/sl-binaries-1.1.4.tar.gz all twelve binaries + checksums + man + completions
release/sl-go_1.1.4-1_amd64.deb
release/packaging/               distribution recipes with checksums filled in
```

That directory is in `.gitignore`, does not exist in a fresh checkout, and
`make clean` removes it — so the source tree stays clean without the build ever
writing outside it.
The archives are reproducible (sorted names, fixed owner and timestamp) and unpack
to `sl-go-1.1.4/`, where a plain `make` builds and `make test` passes.

### 3.10 Handing the binaries to somebody else

Twelve files named `sl-<os>-<arch>` are convenient to run and awkward to pass
around, so `make bundle` (also run by `make packages`) packs them into one
archive:

```sh
make bundle
# -> release/sl-binaries-1.1.4.tar.gz
tar tzf release/sl-binaries-1.1.4.tar.gz
#   sl-binaries-1.1.4/README.txt
#   sl-binaries-1.1.4/SHA256SUMS
#   sl-binaries-1.1.4/bin/sl-linux-amd64        (all twelve)
#   sl-binaries-1.1.4/share/man/man1/sl.1
#   sl-binaries-1.1.4/share/completions/{_sl,sl.bash,sl.fish}
```

`README.txt` inside says what the thing is, installs with one `install` line,
and lists each binary with its format, machine and linkage — including the fact
that the OpenBSD and macOS builds are dynamic and the rest are static.

Checking it is the same two commands as always, because `SHA256SUMS` inside is
relative to the archive root:

```sh
tar xzf sl-binaries-1.1.4.tar.gz && cd sl-binaries-1.1.4
sha256sum -c SHA256SUMS        # 12/12 OK
./bin/sl-linux-amd64 --version
```

The archive is reproducible in the same way as the source tarball (sorted names,
owner 0:0, fixed mtime, `gzip -n`): two runs on the same binaries give the same
file, so its checksum can be published. The binaries inside are byte-identical
to `release/bin/`, so `make check-binaries` still applies to them.

---

## 4. First run

```sh
sl                    # the current directory
sl ~/Downloads
sl -a /usr/include    # everything, including the headers that are hidden by default
sl --only /usr/bin    # only the entries that stand out
```

Read `man sl` (or `sl --help`) for the notation: `/` directory, `->` link,
`*` executable, `0` empty, `1` lone entry in a directory, `<-` hard links,
`X` broken link, `u g s W r x` permission oddities, `<2h` changed recently,
`-5m` read recently, `4.2M` relatively large, `...7` a run of seven names.

---

## 5. Configuration

The first run creates `$XDG_CONFIG_HOME/sl/config`, i.e. normally
`~/.config/sl/config`:

```
$ sl
sl: created /home/you/.config/sl/config (edit it, or run with --no-config)
  ...
```

The file it writes is a **commented-out template**: every setting the program has
is in there, with its default value next to it. That has three consequences worth
knowing:

* the file changes nothing until you uncomment a line, so "no file" and
  "pristine file" behave identically;
* it is the reference documentation for the settings — no need to open the man
  page to see that `relchange` defaults to 10;
* an existing file is **never** overwritten. Edit it freely; if you want a fresh
  copy, delete it and start `sl` again. (`--no-config` reads nothing and creates
  nothing; `--config FILE` names a file that must already exist.)

Format: `key = value` per line, `#` starts a comment, every long command line
option works as a key, and command line options win over the file. Three families
are specific to the file: `style.<name>`, `group.<name>.extra`, and the hook keys.

```ini
# what to show
ignore = 2            # also show C headers (.h) and objects (.o)
relread = 0           # this file system does not update atime

# what to condense
series = 3
prefix = 3

# colours (SGR attributes, without the leading ESC[)
style.dir = 01;34
style.exec = 01;32
style.change = 01;31

# extra suffixes for a group
group.doc.extra = .md .rst
```

Useful keys: `ignore`, `title`, `nosfx`, `prefix`, `series`, `sort`, `summary`,
`expando`, `relsize`, `relchange`, `relread`, `atime`, `k` (1000 or 1024),
`colsep`, `namemin`, `ellipsis`, `color`, `mounts`, `autosearch`, `notable`,
`inline-singleton`, `owner-notes`, `group-mix`, `legacy-classify`,
`legacy-series`, `annotate`, `annotate-match`, and
`style.<name>` / `group.<name>.extra`.

### 5.1 Why not TOML

Because the program is deliberately stdlib-only, and the file describes a
handful of scalars plus two dotted namespaces (`style.*`, `group.*.extra`). TOML
would mean either a dependency (`BurntSushi/toml` or similar, ~10k lines and its
own release cycle) or a hand-written parser of comparable size to the one we have
now — `config.go` is 120 lines and needs no third party. TOML's real strengths
(tables, arrays of tables, typed dates, multi-line strings) are not used by a
config whose longest value is a shell command or a list of suffixes. The flat
format also matches what Unix users already know from `LS_COLORS`, `.gitconfig`
siblings and `sl`'s own command line. If nested configuration is ever needed, the
parser is small enough to revisit; until then, "no dependencies" wins.

### 5.2 Hooks and rules instead of a Tcl startup file

The original let you redefine procedures in `~/.sl.tcl`, including
`perfile` (called for every entry) and `vc-*` (version control marks). A static
binary cannot host an interpreter, so the two things the file was used for have
one replacement each:

* **rules** — force a group by name, no code involved:

  ```ini
  rule = \.(tmp|bak|swp)$ -> ignore
  rule = ^IMG_[0-9]+\.JPG$ -> image
  ```

  Go regular expression, unanchored, case-sensitive unless it starts with
  `(?i)`, matched against the name with the compression wrapper stripped
  (`foo.1.gz` is seen as `foo.1`). First match wins, `ignore` hides the entry.
  `--rule` is the command line form and wins over the file. The full
  description and the list of groups is in `man sl`, section RULES.

* **hooks** — run something per entry and show the result:

  ```ini
  # how long ago was each C file touched, according to git?
  annotate = git log -1 --format=%cr -- "$1" 2>/dev/null
  annotate-match = \.c$
  ```

  One process per entry is expensive; `annotate-batch` runs the command once per
  directory and passes the paths on stdin (one answer line per path):

  ```ini
  annotate-batch = while read -r p; do git log -1 --format=%cr -- "$p"; done
  ```

`$1` is the path of the entry, `$2` its name; the first line of output is
appended to the entry. Because it runs one process per matching entry, keep
`annotate-match` narrow. Other examples:

```ini
# mark files that are modified in git
annotate = git status --porcelain -- "$1" | head -1
annotate-match = .

# line count of every Python file
annotate = wc -l < "$1"
annotate-match = \.py$

# build number next to an Ant build file
annotate = sed -n 's/^build.number=/#/p' "$2"
annotate-match = ^build\.xml$
```

Old habits still work: `~/.sl.tcl` is not executed (that would need Tcl), and
`--startup FILE` prints a note and is ignored.

---

## 6. Uninstall

```sh
sudo apt remove sl-go                        # package
sudo rm /usr/local/bin/sl ~/.local/bin/sl   # manual install (the file is named sl there)
rm -rf ~/.config/sl                         # configuration
```

---

## 7. Troubleshooting

**Colours look wrong / the output is full of `^[[31m`.**
Your terminal does not render ANSI colours, or output goes through a pager that
does not pass them. Use `--no-color`, set `NO_COLOR=1`, or teach
`less` about colour with `LESS=--RAW-CONTROL-CHARS`.

**Names are not coloured although your theme says they should.**
`sl` uses its own palette by default. Export `LS_COLORS` (as `dircolors` does)
and the standard keys `di`, `ln`, `ex`, `or` and the `*.ext` patterns will be
honoured instead.

**`recently read` markers look random.**
Many file systems mount with `noatime`/`relatime`, and on macOS Spotlight
touches access times of everything. Set `relread = 0` in the configuration; the
original had the same problem and documented it as a known bug.

**A directory of two files shows nothing as interesting.**
Relative markers need a handful of entries to mean anything. That is by design
(inherited from the original, which calls this \(lqbecause everybody's special,
nobody's special\(rq).

**`error writing "stdout": broken pipe`** was the Tcl version's way of saying
you pressed `q` in a pager. Not applicable here.

---

## 8. Differences from the Tcl version

* no `~/.sl.tcl`, no `perfile`, no `vc-rcs`/`vc-cvs`; use `--annotate` instead
  (see 5.2);
* the configuration file is created on the first run as a commented template,
  instead of being hand-written (or not written at all) as `~/.sl.tcl`;
* deterministic order (entries sorted by name) instead of the file system's
  `readdir` order;
* `--no-color`, `NO_COLOR`, `LS_COLORS`, `COLUMNS`;
* **the terminal width is read properly.** The Tcl script took it from
  `stty -a` with the regular expression `(\d+)\s+columns`, which matches the BSD
  and macOS spelling (`100 columns`) but not the Linux one (`rows 50; columns
  200`). On Linux it therefore fell back to `$COLUMNS` — a shell variable that is
  normally *not* exported — and then to a hard-coded 80. So the Tcl version lays
  out for an 80-column terminal no matter how wide the window is, which is why it
  shows two columns where this port shows five or six. Measured on `/usr/bin` in
  a 200-column pty: Tcl 2 columns, this port 6; `COLUMNS=200 tclsh sl-1.1.tcl`
  also gives 6, and this port in an 80-column pty gives 2. The layout code and
  the defaults (`NAMEMIN 40`, `COLSEP 1`, `shorten 10`, ...) are the same — only
  the width differs. `--width 80` reproduces the old look exactly;
* Unicode-aware column widths (CJK, combining marks, emoji);
* the Apple II group also knows post-2012 disk image formats
  (`.woz`, `.hdv`, `.d13`, `.edsk`, `.edd`, `.nb2`, `.n35`, `.d16`) and the
  13-sector `.nib` size;
* a directory's contents are classified by their **real type** (`d_type` from
  `readdir`) instead of the Tcl rule "no suffix, so probably a subdirectory",
  which counted `README`, `Makefile` and `LICENSE` as folders. `--legacy-classify`
  restores the old rule, which also restores the old reclassification threshold;
* a **name series** is a name that differs in one run of digits
  (`episode01`/`episode02`) or in a single index letter at the end of the stem
  (`part a`/`part b`); `abc.txt`/`abd.txt` no longer collapse. `--legacy-series`
  restores the Tcl rule;
* `--owner-notes` (on by default) appends `owner=NAME` to files in your home
  directory that belong to somebody else, and `--group-mix` shows `5+3` instead
  of `8` on a directory classified by contents — these two are the original's
  TAP branch, which the Tcl version enabled only when a private directory of the
  author happened to exist; `--experimental` turns both on;
* `--inline-singleton` implements the branch the original left disabled
  (`elseif {0 && $size==1}`): a directory with one entry is shown as
  `src/main.c` instead of `src/1`;
* `--sort name|size|time|atime` orders entries inside a group; the original
  always printed name order. `--sort size` turns on `--relsize 100` by itself,
  keeps directories in name order (their size is a count of entries, not a
  weight) and switches `--prefix` off, because a shared prefix is elided against
  the neighbouring entry;
* the RCS/CVS marks, the Apple resource-fork size fallback and the `touch`-based
  access time hack are not ported.

Everything else — grouping, marks, relative thresholds, series collapsing,
prefix elision, the column layout heuristics — follows the original.
