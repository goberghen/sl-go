#!/bin/sh
# make-bundle.sh - put the finished binaries into one tarball.
#
# Usage:
#   ./packaging/make-bundle.sh            # -> release/sl-binaries-<version>.tar.gz
#   OUTDIR=/tmp/x ./packaging/make-bundle.sh
#
# Why: twelve files named sl-<os>-<arch> are convenient to run and awkward to
# hand over. This packs them, with their checksums, the man page and the shell
# completions, into a single archive that unpacks into sl-<version>/.
#
# The archive is reproducible: members are sorted, owned by 0:0, all carry the
# same fixed mtime, and gzip is called with -n so no build time ends up in the
# header. Running this twice on the same binaries gives the same file.

set -eu
cd "$(dirname "$0")/.."                       # -> sl-go/

VERSION=$(sed -n 's/^const version = "\(.*\)"/\1/p' src/cli.go)
[ -n "$VERSION" ] || { echo "make-bundle: cannot read the version from src/cli.go" >&2; exit 1; }

BIN=release/bin
OUTDIR=${OUTDIR:-release}
NAME=sl-binaries-$VERSION
TARBALL="$OUTDIR/$NAME.tar.gz"
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

ROOT="$STAGE/$NAME"
mkdir -p "$ROOT/bin" "$ROOT/share/man/man1" "$ROOT/share/completions"

# 1. the binaries, exactly as make all built them
count=0
for f in "$BIN"/sl-*; do
	[ -f "$f" ] || { echo "make-bundle: no binaries in $BIN (run 'make all')" >&2; exit 1; }
	cp "$f" "$ROOT/bin/"
	chmod 0755 "$ROOT/bin/$(basename "$f")"
	count=$((count + 1))
done
if [ "$count" -ne 12 ]; then
	echo "make-bundle: expected 12 binaries, found $count" >&2
	exit 1
fi

# 2. what makes the archive usable on a machine without the source tree
cp docs/sl.1 "$ROOT/share/man/man1/sl.1"
cp completions/sl.bash completions/_sl completions/sl.fish "$ROOT/share/completions/"

# 3. checksums of the binaries themselves, relative to the archive root, so that
#    "cd sl-<version> && sha256sum -c SHA256SUMS" works after unpacking
( cd "$ROOT" && sha256sum bin/sl-* > SHA256SUMS )

TITLE="sl $VERSION -- prebuilt binaries"
RULE=$(printf '%*s' "${#TITLE}" '' | tr ' ' '=')

cat > "$ROOT/README.txt" <<EOF
$TITLE
$RULE

sl organizes a directory: it groups entries by kind, collapses name series,
marks what stands out from the rest of the directory (large, recently changed,
empty, world-writable, broken link, ...) and prints a summary. A Go port of the
Tcl script sl 1.1.3 by Thomas A. Phelps.

This is not a replacement for ls: it summarises rather than lists.

Contents
--------
bin/sl-<os>-<arch>          twelve static binaries, one per platform
SHA256SUMS                  checksums of the binaries in bin/
share/man/man1/sl.1         manual page
share/completions/          bash, zsh and fish completion files

Check what you received
-----------------------
    sha256sum -c SHA256SUMS          # inside the unpacked sl-$VERSION/

Install
-------
    install -m 0755 bin/sl-linux-amd64 /usr/local/bin/sl
    install -D -m 0644 share/man/man1/sl.1 /usr/local/share/man/man1/sl.1
    install -D -m 0644 share/completions/_sl /usr/local/share/zsh/site-functions/_sl

    (or: make install from the source tarball, sl-go-$VERSION.tar.gz)

The binary knows nothing about completions: they are plain files next to it,
which is why they are here and not behind a flag.

Platforms
---------
EOF
# one short line per binary: format, machine, and whether it needs a library
for f in "$ROOT"/bin/sl-*; do
	desc=$(file -b "$f" 2>/dev/null || echo "")
	machine=$(printf '%s' "$desc" |
		sed -e 's/PowerPC or cisco 7500/PowerPC/' -e 's/Intel 80386/i386/' |
		sed -n 's/.*\(x86-64\|x86_64\|aarch64\|arm64\|i386\|ARM\|PowerPC\).*/\1/p' | head -1)
	[ -n "$machine" ] || machine="-"
	case "$desc" in
		ELF*64-bit*) kind="ELF 64-bit" ;;
		ELF*32-bit*) kind="ELF 32-bit" ;;
		Mach-O*64-bit*|Mach-O*arm64*) kind="Mach-O 64-bit" ;;
		Mach-O*) kind="Mach-O" ;;
		*) kind=$(printf '%s' "$desc" | cut -d, -f1 | cut -c1-20) ;;
	esac
	case "$kind" in
		Mach-O*) link="links libSystem" ;;   # Go on macOS always does
		*) link=$(printf '%s' "$desc" | grep -o 'statically linked\|dynamically linked' | head -1) ;;
	esac
	[ -n "$link" ] || link="unknown linkage"
	printf '    %-22s %-10s %s, %s\n' "$(basename "$f")" "$machine" "$kind" "$link" \
		>> "$ROOT/README.txt"
done
cat >> "$ROOT/README.txt" <<EOF

See also
--------
man sl            after installing the man page
sl --help         every option
INSTALL.md        in the source tarball: building, packaging, troubleshooting

License: GPL-3.0-or-later (the original Tcl script is GPL too).
EOF

# 4. one file, reproducible
mkdir -p "$OUTDIR"
rm -f "$TARBALL"
tar --sort=name --owner=0 --group=0 --numeric-owner \
	--mtime='UTC 2026-01-01' -cf - -C "$STAGE" "$NAME" |
	gzip -9n > "$TARBALL"

echo "make-bundle: $TARBALL ($count binaries, $(wc -c < "$TARBALL") bytes)"
