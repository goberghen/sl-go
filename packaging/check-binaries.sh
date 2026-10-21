#!/bin/sh
# check-binaries.sh BINARY... - verify that each built binary has the linkage
# its platform requires, and that none of them is more dynamic than it should be.
#
# Usage:
#   ./packaging/check-binaries.sh release/bin/sl-linux-amd64 release/bin/sl-freebsd-amd64
#   ALPINE_ROOTFS=/tmp/alpine ./packaging/check-binaries.sh release/bin/sl-linux-amd64
#
# Why this exists, and why it is not "check for static":
#
#   * Linux, FreeBSD and NetBSD: the Go runtime issues raw system calls, so the
#     binary has no interpreter and no NEEDED entries. Being static is what lets
#     ONE Linux file be shipped to glibc and musl systems alike.
#   * OpenBSD: Go's port calls libc for its system calls ("System calls are
#     implemented in libc/libpthread" -- runtime/sys_openbsd_amd64.s), so the
#     link is dynamic by design and cannot be changed with a flag (-linkmode
#     internal does not help: the calls are still there). Expected: exactly
#     libc.so and libpthread.so, both part of the base system.
#   * macOS: the same story with libSystem, which Apple requires.
#
# So the check is "each binary is as linked as its platform demands": a
# regression on FreeBSD is caught as loudly as one on Linux, and the two
# platforms that are dynamic by design are checked against their expected shape
# instead of being skipped quietly.

set -eu

[ $# -ge 1 ] || { echo "usage: check-binaries.sh BINARY..." >&2; exit 2; }
fail=0

elf_os() { readelf -h "$1" 2>/dev/null | sed -n 's/^ *OS\/ABI: *//p' | head -1; }
needed() { readelf -d "$1" 2>/dev/null | sed -n 's/.*Shared library: \[\(.*\)\]/\1/p'; }
has_interp() { readelf -S "$1" 2>/dev/null | grep -q '\.interp'; }

is_macho() {
	magic=$(od -An -tx1 -N4 "$1" 2>/dev/null | tr -d ' \n')
	case "$magic" in
	cffaedfe | cefaedfe | cafebabe | bebafeca) return 0 ;;
	esac
	return 1
}

for bin in "$@"; do
	[ -f "$bin" ] || {
		echo "check-binaries: $bin: no such file"
		exit 1
	}

	os=$(elf_os "$bin")
	if [ -n "$os" ]; then
		case "$os" in
		"UNIX - System V" | "UNIX - FreeBSD" | "UNIX - NetBSD")
			if [ "$(needed "$bin" | wc -l)" -ne 0 ] || has_interp "$bin"; then
				echo "check-binaries: $bin: FAIL, should be static on $os:"
				needed "$bin" | sed 's/^/    NEEDED /'
				has_interp "$bin" && echo "    has an interpreter (.interp)"
				fail=1
			else
				echo "check-binaries: $bin: static, no NEEDED entries"
			fi
			;;
		"UNIX - OpenBSD")
			list=$(needed "$bin" | sort | tr '\n' ' ')
			case "$list" in
			"libc.so libpthread.so ")
				echo "check-binaries: $bin: dynamic against libc/libpthread, as expected on OpenBSD"
				has_interp "$bin" || {
					echo "check-binaries: $bin: FAIL, dynamic but has no interpreter"
					fail=1
				}
				;;
			"")
				echo "check-binaries: $bin: note: now static (this used to be dynamic);"
				echo "check-binaries:   if that is deliberate, update INSTALL.md 3.6"
				;;
			*)
				echo "check-binaries: $bin: FAIL, unexpected OpenBSD linkage: $list"
				fail=1
				;;
			esac
			;;
		*)
			echo "check-binaries: $bin: FAIL, unknown ELF OS/ABI: $os"
			fail=1
			;;
		esac
		continue
	fi

	if is_macho "$bin"; then
		if grep -aq '/usr/lib/libSystem' "$bin"; then
			echo "check-binaries: $bin: Mach-O, linked against libSystem, as required on macOS"
		else
			echo "check-binaries: $bin: FAIL, Mach-O without libSystem"
			fail=1
		fi
		continue
	fi

	echo "check-binaries: $bin: FAIL, neither ELF nor Mach-O"
	fail=1
done

if [ -n "${ALPINE_ROOTFS:-}" ]; then
	# An x86_64 rootfs can only execute x86_64 binaries; the others are the same
	# static file for another machine and have nothing to prove here.
	host=$(uname -m)
	for bin in "$@"; do
		[ -n "$(elf_os "$bin")" ] || continue
		[ "$(elf_os "$bin")" = "UNIX - System V" ] || continue
		machine=$(readelf -h "$bin" 2>/dev/null | sed -n 's/^ *Machine: *//p')
		case "$host:$machine" in
		x86_64:*X86-64* | aarch64:*AArch64* | arm64:*AArch64* | riscv64:*RISC-V*) ;;
		*)
			echo "check-binaries: $bin: $machine, not $host, skipped the musl run"
			continue
			;;
		esac
		cp "$bin" "$ALPINE_ROOTFS/sl-check" 2>/dev/null || {
			echo "check-binaries: cannot write to $ALPINE_ROOTFS (run as root?)" >&2
			exit 1
		}
		if out=$(sudo -n chroot "$ALPINE_ROOTFS" /sl-check --version 2>&1 | head -1); then
			echo "check-binaries: $bin: runs on musl: $out"
		else
			echo "check-binaries: $bin: FAIL under musl: $out"
			fail=1
		fi
		rm -f "$ALPINE_ROOTFS/sl-check"
	done
else
	echo "check-binaries: ALPINE_ROOTFS not set, skipped the musl run"
fi

exit $fail
