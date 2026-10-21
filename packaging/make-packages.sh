#!/bin/sh
# make-packages.sh - stage every distribution recipe with the real values.
#
# Usage:
#   ./packaging/make-packages.sh          # recipes -> release/packaging/,
#                                         # checksums -> release/SHA256SUMS
#   OUT=/tmp/p ./packaging/make-packages.sh
#
# What it does, and why it is a script rather than a checklist in INSTALL.md:
# every recipe below needs the release tarball and its checksum, and hand-
# editing a checksum into five files is how packages end up building from the
# previous version. The recipes in packaging/ keep the placeholders
# (@VERSION@, @SHA256@, @SHA512@, @DATE@); this script fills them in and copies
# the result next to the tarball, ready to be submitted to the respective
# distribution. Nothing here needs network access or a package manager: the
# only tools used are make, sha256sum/sha512sum, sed and dpkg-deb.

set -eu
cd "$(dirname "$0")/.."                       # -> sl-go/

VERSION=$(sed -n 's/^const version = "\(.*\)"/\1/p' src/cli.go)
[ -n "$VERSION" ] || { echo "make-packages: cannot read the version from src/cli.go" >&2; exit 1; }
OUT=${OUT:-release/packaging}
TARBALL="release/sl-go-$VERSION.tar.gz"

echo "make-packages: version $VERSION"
make dist >/dev/null
[ -f "$TARBALL" ] || { echo "make-packages: $TARBALL is missing" >&2; exit 1; }

SHA256=$(sha256sum "$TARBALL" | cut -d' ' -f1)
SHA512=$(sha512sum "$TARBALL" | cut -d' ' -f1)
# Guix writes checksums in base32, not hex (52 characters, no padding). Doing
# the conversion here is the difference between a recipe that can be submitted
# and one that has to be finished by hand.
if command -v python3 >/dev/null 2>&1; then
	BASE32=$(printf '%s' "$SHA256" | python3 -c 'import sys,base64; print(base64.b32encode(bytes.fromhex(sys.stdin.read().strip())).decode().lower().rstrip("="))')
else
	BASE32="@BASE32@"
	echo "make-packages: python3 not found, the Guix checksum stays a placeholder" >&2
fi
DATE=$(LC_ALL=C date '+%a %b %d %Y')

mkdir -p "$OUT"
fill() {                                        # fill SRC DEST
	sed -e "s/@VERSION@/$VERSION/g" \
	    -e "s/@SHA256@/$SHA256/g" \
	    -e "s/@SHA512@/$SHA512/g" \
	    -e "s/@DATE@/$DATE/g" \
	    -e "s/@BASE32@/$BASE32/g" \
	    "$1" > "$2"
}

fill packaging/sl-go.spec  "$OUT/sl-go.spec"
fill packaging/PKGBUILD    "$OUT/PKGBUILD"
fill packaging/APKBUILD    "$OUT/APKBUILD"
fill packaging/sl-go.rb    "$OUT/sl-go.rb"
fill packaging/freebsd/Makefile "$OUT/freebsd-Makefile"
cp packaging/freebsd/pkg-descr packaging/freebsd/pkg-plist "$OUT/"
fill packaging/openbsd/Makefile "$OUT/openbsd-Makefile"
cp packaging/nix/default.nix "$OUT/nix-default.nix"
cp flake.nix "$OUT/flake.nix"
fill packaging/guix/sl-go.scm "$OUT/sl-go.scm"
cp packaging/openbsd/pkg/DESCR "$OUT/openbsd-pkg-DESCR"
cp packaging/openbsd/pkg/PLIST "$OUT/openbsd-pkg-PLIST"

# The linkage check is part of "staging a release", not an optional nicety: a
# Linux binary that silently grew a libc dependency would stop working on musl.
if ls release/bin/* >/dev/null 2>&1; then
	./packaging/check-binaries.sh release/bin/*
else
	echo "make-packages: no binaries, skipped the linkage check (run make all)" >&2
fi

# The Debian package is built here rather than copied, so that a stale .deb
# cannot be mistaken for the current one. It lands in release/, next to the
# tarball and the binaries -- not in release/packaging/, which holds recipes
# (text) and nothing else. One artifact, one place.
if command -v dpkg-deb >/dev/null 2>&1; then
	ARCH=${ARCH:-$(dpkg --print-architecture 2>/dev/null || echo amd64)} \
		OUT=release ./packaging/build-deb.sh >/dev/null
else
	echo "make-packages: dpkg-deb not found, skipping the .deb" >&2
fi

# Checksums for everything that is being handed out.
# The checksum file describes release/ as a whole, so it is always written
# there, even when OUT points somewhere else. Paths in it are relative to
# release/, so that `cd release && sha256sum -c SHA256SUMS` just works. The recipes in
# packaging/ are not listed: they are not artifacts, and a checksum file that
# contains a checksum of itself (which is what used to happen) is nonsense.
(
	cd release
	sha256sum sl-go-*.tar.gz sl-go_*.deb bin/sl-*
) > release/SHA256SUMS

echo "make-packages: tarball  $TARBALL"
echo "make-packages: sha256   $SHA256"
echo "make-packages: recipes in $OUT"
ls -1 "$OUT"
echo "make-packages: checksums in release/SHA256SUMS"
