# FreeBSD port skeleton (sysutils/sl-go).
#
#   # in a ports tree: mkdir -p sysutils/sl-go && cp -r packaging/freebsd/* sysutils/sl-go/
#   make makesum && make install
#
# The original Tcl script is what ports/sysutils/sl-ls packaged; that port is
# long dead (its HOMEPAGE no longer resolves), so this one is a fresh port
# rather than an update of it.  The same binary name is used, and CONFLICTS is
# how a port expresses "not at the same time as the other one".

PORTNAME=	sl-go
DISTVERSION=	@VERSION@
CATEGORIES=	sysutils
MASTER_SITES=	https://example.invalid/sl-go/

MAINTAINER=	sl@example.invalid
COMMENT=	More useful ls: group files by purpose and mark interesting ones
WWW=		https://web.archive.org/web/2016/http://practicalthought.com/sl/

LICENSE=	GPLv3+
LICENSE_FILE=	${WRKSRC}/LICENSE

USES=		go:modules
GO_MODULE=	sl
CONFLICTS_INSTALL=	sl-ls

PLIST_FILES=	bin/sl \
		share/man/man1/sl.1.gz \
		share/bash-completion/completions/sl \
		share/zsh/site-functions/_sl \
		share/fish/vendor_completions.d/sl.fish \
		share/doc/sl-go/README.md \
		share/doc/sl-go/INSTALL.md

do-install:
	${INSTALL_PROGRAM} ${WRKSRC}/sl ${STAGEDIR}${PREFIX}/bin/sl
	${INSTALL_MAN} ${WRKSRC}/docs/sl.1 ${STAGEDIR}${PREFIX}/share/man/man1/sl.1
	${INSTALL_DATA} ${WRKSRC}/completions/sl.bash ${STAGEDIR}${PREFIX}/share/bash-completion/completions/sl
	${INSTALL_DATA} ${WRKSRC}/completions/_sl ${STAGEDIR}${PREFIX}/share/zsh/site-functions/_sl
	${INSTALL_DATA} ${WRKSRC}/completions/sl.fish ${STAGEDIR}${PREFIX}/share/fish/vendor_completions.d/sl.fish
	@${MKDIR} ${STAGEDIR}${DOCSDIR}
	${INSTALL_DATA} ${WRKSRC}/README.md ${WRKSRC}/INSTALL.md ${STAGEDIR}${DOCSDIR}

.include <bsd.port.mk>
