GO      ?= go
GOFLAGS_BUILD := -trimpath -ldflags "-s -w"
# Cross-compiled by `make all`. All of them are pure Go (CGO_ENABLED=0), so they
# are static: the Linux ones run on glibc and musl alike (see INSTALL.md 3.5),
# the BSD ones have no runtime dependencies either. Every binary is named after
# the platform it was built for -- sl-linux-amd64 and not a bare `sl`: the host
# build is the same file as one of these (byte for byte, same flags), so a bare
# name would only add a second copy whose name says nothing about the target.
# Turning sl-linux-amd64 into `sl` is the installer's job: `make install` and
# every package recipe put it in $PATH as sl.
TARGETS := linux/amd64 linux/arm64 linux/386 linux/arm linux/ppc64 linux/ppc64le \
           darwin/amd64 darwin/arm64 freebsd/amd64 \
           openbsd/amd64 openbsd/arm64 netbsd/amd64

# Sources live in src/; everything that a build produces goes to release/,
# which is inside the source tree, empty by default and wiped by `make clean`.
# Nothing is ever written above sl-go/.
SRC ?= ./src
OUT ?= release
BIN := $(OUT)/bin

VERSION := $(shell sed -n 's/^const version = "\(.*\)"/\1/p' src/cli.go)
# The host build is just one of the TARGETS, so it is named like one.
HOSTOS   := $(shell $(GO) env GOOS 2>/dev/null)
HOSTARCH := $(shell $(GO) env GOARCH 2>/dev/null)
HOSTBIN  := $(BIN)/sl-$(HOSTOS)-$(HOSTARCH)
TARBALL := $(OUT)/sl-go-$(VERSION).tar.gz
# everything that belongs in a source tarball; release/ itself is not listed
SRCFILES := Makefile README.md INSTALL.md LICENSE go.mod .gitignore flake.nix src docs bench packaging \
            completions tools

.PHONY: default all build share test vet bench bench-ab verify check-binaries completions clean install uninstall deb packages bundle man layout dist

# `make` = build the binary and refresh the source tarball.
default: build dist

# The completion files as shipped: plain copies of the committed files, next to
# the binaries. Both `build` and `all` need them, `all` no longer goes through
# `build` (it builds the host target as part of its own loop).
share:
	@mkdir -p $(OUT)/share/completions
	@cp completions/sl.bash completions/_sl completions/sl.fish $(OUT)/share/completions/

build: share
	@test -n "$(HOSTOS)" -a -n "$(HOSTARCH)" || { \
		echo "build: cannot determine the host GOOS/GOARCH -- is $(GO) on PATH?"; exit 1; }
	@mkdir -p $(BIN)
	CGO_ENABLED=0 GOOS=$(HOSTOS) GOARCH=$(HOSTARCH) $(GO) build $(GOFLAGS_BUILD) -o $(HOSTBIN) $(SRC)

# Shell completion files. They are plain files in the tree, not something sl
# prints on demand: generated once here from the option table in src/cli.go, and
# kept honest by TestCompletionsMatchOptions.
completions:
	$(GO) run ./tools/completions -src src/cli.go -out completions

all: dist share
	@mkdir -p $(BIN)
	@for t in $(TARGETS); do \
		os=`echo $$t | cut -d/ -f1`; arch=`echo $$t | cut -d/ -f2`; \
		echo "build $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build $(GOFLAGS_BUILD) -o $(BIN)/sl-$$os-$$arch $(SRC) || exit 1; \
	done

# dist refreshes release/sl-go-<version>.tar.gz from the current sources.
# Sorted, fixed owner, no mtime of the working copy: the same sources always
# produce the same archive.
dist:
	@mkdir -p $(OUT)
	@rm -f $(TARBALL)
	tar --sort=name --owner=0 --group=0 --numeric-owner --mtime='UTC 2026-01-01' \
		-czf $(TARBALL) --transform 's,^,sl-go-$(VERSION)/,' $(SRCFILES)
	@echo "dist: $(TARBALL)"

test:
	$(GO) test $(SRC)

# Two ways to measure, see bench/README.md.
#   make bench    -- in-process, no process start, compares each hand-written
#                    matcher with the regular expression it replaced
#   make bench-ab -- wall clock of two real binaries, interleaved; windows/startup
#                    included; A=... B=... DIRS="..."
bench:
	go test $(SRC) -run XXX -bench . -benchtime 80x -count 5

bench-ab:
	@test -n "$(A)" || { echo 'usage: make bench-ab A=old/sl B=new/sl DIRS="/usr/bin /etc"'; exit 2; }
	go run ./bench -a $(A) $(if $(B),-b $(B)) -n $(or $(N),30) $(DIRS)

# There is no "static build" to ask for: CGO_ENABLED=0 is the only build mode, so
# `make all` does not have a static variant. What is worth having is a check,
# because linkage is a property that can change silently (a cgo dependency, a
# new Go port, a different linker):
#
#   make check-binaries   verify every binary already in $(BIN) -- Linux,
#                         FreeBSD and NetBSD must be static, OpenBSD and macOS
#                         must be dynamically linked against the system libc,
#                         which is what their Go ports do by design (see
#                         INSTALL.md 3.6). Fast: no build.
#   make verify           `make all` and then the check above.
#
# With ALPINE_ROOTFS=/path/to/alpine-minirootfs the check also runs the Linux
# binaries inside Alpine, i.e. against musl.
check-binaries:
	@set -- $(BIN)/*; \
	for f in "$$@"; do \
		test -f "$$f" || { echo "check-binaries: no binaries in $(BIN), run \`make all\` first"; exit 1; }; \
	done; \
	./packaging/check-binaries.sh "$$@"

verify: all check-binaries

vet:
	$(GO) vet $(SRC)
	gofmt -l $(SRC)

# The Debian package is built by packaging/build-deb.sh; this target is a thin
# wrapper so that `make deb` and `./packaging/build-deb.sh` do the same thing.
deb:
	@ARCH=$(or $(ARCH),$(shell dpkg --print-architecture 2>/dev/null || echo amd64)) \
	OUT=$(OUT) ./packaging/build-deb.sh

# Stage every distribution recipe (deb, rpm, PKGBUILD, APKBUILD, Homebrew,
# FreeBSD) with the tarball's checksum into release/packaging/.
packages: bundle
	@./packaging/make-packages.sh

# bundle puts the twelve finished binaries into one reproducible tarball,
# release/sl-binaries-<version>.tar.gz, together with their checksums, the man
# page and the completions. Twelve loose files are convenient to run and
# awkward to hand over; the archive is what goes to somebody else.
bundle: share
	@./packaging/make-bundle.sh

man:
	@mandoc -Tlint docs/sl.1 2>/dev/null || echo "man: mandoc not installed, skipping lint"

install: build
	install -D -m 0755 $(HOSTBIN) /usr/local/bin/sl
	install -D -m 0644 docs/sl.1 /usr/local/share/man/man1/sl.1
	install -D -m 0644 completions/sl.bash /usr/local/share/bash-completion/completions/sl
	install -D -m 0644 completions/_sl /usr/local/share/zsh/site-functions/_sl
	install -D -m 0644 completions/sl.fish /usr/local/share/fish/vendor_completions.d/sl.fish

uninstall:
	rm -f /usr/local/bin/sl /usr/local/share/man/man1/sl.1
	rm -f /usr/local/share/bash-completion/completions/sl
	rm -f /usr/local/share/zsh/site-functions/_sl
	rm -f /usr/local/share/fish/vendor_completions.d/sl.fish

# Print the layout, handy after a checkout.
layout:
	@echo "sources:  $(SRC)"
	@echo "docs:     docs/sl.1, README.md, INSTALL.md"
	@echo "shell:    completions/{sl.bash,_sl,sl.fish}  (make completions)"
	@echo "builds:   $(BIN)"
	@echo "packages: $(OUT)/  (sl-go-$(VERSION).tar.gz, sl-go_$(VERSION)-1_<arch>.deb)"
	@echo "recipes:  packaging/  (deb, rpm, PKGBUILD, APKBUILD, Homebrew, FreeBSD)"
	@echo "bench:    bench/  (make bench, make bench-ab)"

clean:
	rm -rf $(OUT)
