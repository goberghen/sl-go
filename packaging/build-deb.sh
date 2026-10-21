#!/bin/sh
# build-deb.sh - build the sl-go Debian package without debhelper or lintian.
#
# Usage:
#   ./packaging/build-deb.sh            # host architecture
#   ARCH=arm64 ./packaging/build-deb.sh # cross-compiled
#   ARCH=all   ./packaging/build-deb.sh # every binary in one package
#   OUT=/tmp/x ./packaging/build-deb.sh # where the .deb is written (default release/)
#
# `make deb` is a thin wrapper around this script, so the two are equivalent:
# the script is kept separate because Debian packaging is usually driven by a
# script (debhelper, pbuilder, CI images all expect one) and because it can be
# run on a machine where make is not installed. All it needs is go, dpkg-deb,
# gzip and install.
#
# The package is called sl-go (the Go port) and installs /usr/bin/sl and
# /usr/share/man/man1/sl.1.gz. Debian's own "sl" package (the steam locomotive
# joke) wants the very same paths, so this package declares
#
#	Conflicts: sl   -- the two cannot be installed at the same time
#	Replaces:  sl   -- and swapping one for the other is a normal upgrade
#	Provides:  sl   -- anything that depends on "sl" is satisfied
#
# which gives the "mutually exclusive" behaviour: the package manager refuses
# to have both and asks before replacing one with the other.

set -eu

cd "$(dirname "$0")/.."          # -> sl-go/

VERSION=$(sed -n 's/^const version = "\(.*\)"/\1/p' src/cli.go)
[ -n "$VERSION" ] || { echo "build-deb: cannot read the version from src/cli.go" >&2; exit 1; }

REVISION=${REVISION:-1}
ARCH=${ARCH:-$(dpkg --print-architecture 2>/dev/null || echo amd64)}
NAME=sl-go
DEB_VERSION="${VERSION}-${REVISION}"
OUT=${OUT:-release}              # inside sl-go/, empty by default, wiped by make clean
BIN="$OUT/bin"
MAINTAINER=${MAINTAINER:-"goberghen & Arena.ai agents <sl@example.invalid>"}

# ---------------------------------------------------------------- build binaries
case "$ARCH" in
	all)
		make all >/dev/null
		BINARCH=amd64
		;;
	*)
		: "${GOOS:=linux}"
		case "$ARCH" in
			amd64) GOARCH=amd64; BINARCH=amd64 ;;
			arm64) GOARCH=arm64; BINARCH=arm64 ;;
			i386)  GOARCH=386;   BINARCH=386 ;;
			armhf) GOARCH=arm;   BINARCH=arm ;;
			*) echo "build-deb: unknown architecture $ARCH" >&2; exit 1 ;;
		esac
		export GOARCH
		echo "build-deb: building $GOOS/$GOARCH"
		mkdir -p "$BIN"
		CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
			go build -trimpath -ldflags "-s -w" -o "$BIN/sl-linux-$GOARCH" ./src
		;;
esac

[ -f "$BIN/sl-linux-$BINARCH" ] || { echo "build-deb: $BIN/sl-linux-$BINARCH is missing" >&2; exit 1; }

# ---------------------------------------------------------------- stage
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT INT TERM

install -D -m 0755 "$BIN/sl-linux-$BINARCH" "$STAGE/usr/bin/sl"

if [ "$ARCH" = all ]; then
	# Ship the amd64 image as /usr/bin/sl and keep the others as data, so that
	# an "Architecture: all" build still contains every binary.
	mkdir -p "$STAGE/usr/lib/sl-go"
	for a in amd64 arm64; do
		[ -f "$BIN/sl-linux-$a" ] && install -m 0755 "$BIN/sl-linux-$a" "$STAGE/usr/lib/sl-go/sl-linux-$a"
	done
fi

install -D -m 0644 docs/sl.1 "$STAGE/usr/share/man/man1/sl.1"
gzip -9n "$STAGE/usr/share/man/man1/sl.1"

# shell completions, in the directories each shell looks at
install -D -m 0644 completions/sl.bash "$STAGE/usr/share/bash-completion/completions/sl"
# Debian's zsh does NOT put /usr/share/zsh/site-functions in $fpath: the
# directory exists, but the package-managed location is vendor-completions.
# A file installed to site-functions is silently ignored -- which is exactly
# what used to happen here (measured: `sl -<TAB>` completed nothing until the
# file was moved). Other distributions use site-functions, Debian does not.
install -D -m 0644 completions/_sl     "$STAGE/usr/share/zsh/vendor-completions/_sl"
install -D -m 0644 completions/sl.fish "$STAGE/usr/share/fish/vendor_completions.d/sl.fish"

DOCDIR="$STAGE/usr/share/doc/$NAME"
install -D -m 0644 README.md "$DOCDIR/README.md"
install -D -m 0644 INSTALL.md "$DOCDIR/INSTALL.md"
install -D -m 0644 LICENSE "$DOCDIR/copyright"

# copyright in the machine-readable Debian format
{
	echo "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/"
	echo "Upstream-Name: sl-go"
	echo "Source: https://web.archive.org/web/2016/http://practicalthought.com/sl/ (original Tcl script)"
	echo
	echo "Files: *"
	echo "Copyright: 2011-2012 Thomas A. Phelps (original Tcl script)"
	echo "           2026 goberghen & Arena.ai agents (Go port)"
	echo "License: GPL-3+"
	echo " This program is free software: you can redistribute it and/or modify it"
	echo " under the terms of the GNU General Public License as published by the Free"
	echo " Software Foundation, either version 3 of the License, or (at your option)"
	echo " any later version."
	echo " ."
	echo " This program is distributed in the hope that it will be useful, but WITHOUT"
	echo " ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or"
	echo " FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for"
	echo " more details."
	echo " ."
	echo " On Debian systems the full text of the GNU General Public License version 3"
	echo " can be found in /usr/share/common-licenses/GPL-3."
} > "$DOCDIR/copyright"

{
	echo "$NAME ($DEB_VERSION) unstable; urgency=medium"
	echo
	echo "  * Go port of Thomas A. Phelps' Tcl script sl (v1.1.3, rev 1.89, 2012);"
	echo "    the port continues that version line, hence 1.1.4."
	echo "  * Static binary, installed as /usr/bin/sl; Conflicts/Replaces/Provides: sl."
	echo "  * Grouping, marks, relative thresholds and layout follow the original."
	echo "  * A directory's contents are classified by their real type (d_type)"
	echo "    instead of \"no suffix means subdirectory\"; name series need one run"
	echo "    of digits (or an index letter) instead of any single difference."
	echo "  * Both original rules remain available as"
	echo "    --legacy-classify and --legacy-series."
	echo "  * The user configuration file is created on first run."
	echo
	echo " -- $MAINTAINER  $(date -R)"
} > "$DOCDIR/changelog.Debian"
gzip -9n "$DOCDIR/changelog.Debian"

SIZE=$(du -sk "$STAGE/usr" | cut -f1)

# ---------------------------------------------------------------- control
mkdir -p "$STAGE/DEBIAN"
SRC_URL="https://web.archive.org/web/2016/http://practicalthought.com/sl/"
cat > "$STAGE/DEBIAN/control" <<EOF
Package: $NAME
Version: $DEB_VERSION
Architecture: $ARCH
Maintainer: $MAINTAINER
Installed-Size: $SIZE
Section: utils
Priority: optional
Conflicts: sl
Replaces: sl
Provides: sl
Homepage: $SRC_URL
Description: more useful ls: group files by purpose and point out interesting ones
 sl groups files by purpose (headers, source code, web pages, documents,
 images, audio, video, fonts, archives, Apple II disk images, ...), marks the
 ones that stand out in the current directory (relatively large, recently
 changed or read, empty, unreadable, world-writable, broken link, odd
 permissions, part of a name series), condenses runs of numbered names, and
 lays the result out in columns that fit the terminal.
 .
 This is a Go port of the Tcl script by Thomas A. Phelps (2012), whose home
 page is offline; it is reproduced faithfully, with the Tcl hooks replaced by
 external commands (--annotate) and configuration moved to
 \$XDG_CONFIG_HOME/sl/config, which is created on the first run.
 .
 The binary and man page are installed under the name "sl", so scripts and
 habits keep working. Debian's own "sl" (the steam locomotive) is declared
 as a conflict: install one or the other, not both.
EOF

# conffiles: none, configuration is per user in $XDG_CONFIG_HOME

# ---------------------------------------------------------------- build
mkdir -p "$OUT"
DEB="$OUT/${NAME}_${DEB_VERSION}_${ARCH}.deb"
dpkg-deb --build --root-owner-group "$STAGE" "$DEB" >/dev/null
echo "build-deb: $DEB"
dpkg-deb -I "$DEB" | sed 's/^/  /'
