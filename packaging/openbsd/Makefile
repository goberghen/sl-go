# OpenBSD port skeleton (sysutils/sl-go).
#
#   mkdir -p /usr/ports/sysutils/sl-go && cp -R packaging/openbsd/* /usr/ports/sysutils/sl-go/
#   cd /usr/ports/sysutils/sl-go && make makesum && make install
#
# The Tcl original had an OpenBSD port once (sysutils/sl-ls): its HOMEPAGE,
# practicalthought.com/sl/, no longer resolves, and the port went with it.
# This is a fresh port of the Go rewriting, with the same binary name, hence
# CONFLICTS against the old package.
#
# The program itself needed one fix to build here at all: OpenBSD calls the
# timestamp fields of struct stat Mtim/Atim, while FreeBSD, NetBSD and Darwin
# call them Mtimespec/Atimespec. That is now st_time_openbsd.go, and
# st_time_bsd.go holds the other three.

COMMENT =	usefully organise directories and point out interesting files
VERSION =	@VERSION@
DISTNAME =	sl-go-${VERSION}
CATEGORIES =	sysutils

HOMEPAGE =	https://web.archive.org/web/2016/http://practicalthought.com/sl/

MAINTAINER =	sl@example.invalid

# GPL-3.0-or-later
PERMIT_PACKAGE =	Yes

# The binary is pure Go and static; libc is listed because the Go linker adds
# it to the dynamic table on OpenBSD when cgo is off, but nothing is loaded.
WANTLIB +=	c

MODULES =	lang/go

# The module path in go.mod. A real submission would either publish the module
# or carry the source in the distfile and set MODGO_MODULES/MODGO_MODFILES
# accordingly; the skeleton keeps the two lines a porter has to touch.
MODGO_MODNAME =	sl
MODGO_VERSION =	v${VERSION}

NO_TEST =	Yes

CONFLICTS =	sl-ls-*

ALL_TARGET =	sl

do-install:
	${INSTALL_PROGRAM} ${WRKSRC}/sl ${PREFIX}/bin/sl
	${INSTALL_MAN} ${WRKSRC}/docs/sl.1 ${PREFIX}/man/man1/sl.1
	${INSTALL_DATA_DIR} ${PREFIX}/share/bash-completion/completions
	${INSTALL_DATA_DIR} ${PREFIX}/share/zsh/site-functions
	${INSTALL_DATA_DIR} ${PREFIX}/share/fish/vendor_completions.d
	${INSTALL_DATA} ${WRKSRC}/completions/sl.bash ${PREFIX}/share/bash-completion/completions/sl
	${INSTALL_DATA} ${WRKSRC}/completions/_sl ${PREFIX}/share/zsh/site-functions/_sl
	${INSTALL_DATA} ${WRKSRC}/completions/sl.fish ${PREFIX}/share/fish/vendor_completions.d/sl.fish
	${INSTALL_DATA_DIR} ${PREFIX}/share/doc/sl-go
	${INSTALL_DATA} ${WRKSRC}/README.md ${WRKSRC}/INSTALL.md \
		${PREFIX}/share/doc/sl-go/

.include <bsd.port.mk>
