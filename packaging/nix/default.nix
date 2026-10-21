# Nix derivation for sl.  Use it in one of three ways:
#
#   nix build .#sl-go                  # through flake.nix in the repo root
#   nix-build packaging/nix            # plain, no flakes
#   nix profile install .#sl-go
#
# There are no dependencies at all (the program is standard-library-only), so
# there is no vendorHash to maintain and no FOD: `vendorHash = null` is correct
# and stays correct as long as go.mod has no requires.
{
  lib,
  buildGoModule,
  installShellFiles,
}:

buildGoModule rec {
  pname = "sl-go";
  version = "1.1.4";

  # The whole tree: buildGoModule only wants the Go sources, but the manual
  # page and the licences live next to them, and `src = ../..` keeps the
  # derivation and the repository in step without a second copy of the code.
  src = lib.cleanSource ../..;

  vendorHash = null;

  # go.mod declares `module sl`; the main package is in ./src.
  subPackages = [ "src" ];
  # No -X here: the version is a Go constant in cli.go, not a variable, so
  # there is nothing for the linker to override (and a wrong -X is silently
  # ignored, which would be worse).
  ldflags = [
    "-s"
    "-w"
  ];

  nativeBuildInputs = [ installShellFiles ];

  # CGO_ENABLED=0 is what buildGoModule does by default, which is exactly what
  # this program wants: no libc, so the same store path works on musl-based
  # systems too.
  postInstall = ''
    installManPage docs/sl.1
    installShellCompletion --bash completions/sl.bash
    installShellCompletion --zsh completions/_sl
    installShellCompletion --fish completions/sl.fish
    install -Dm644 README.md INSTALL.md LICENSE -t $out/share/doc/${pname}
  '';

  # The test suite is pure Go and needs nothing outside the build sandbox —
  # except the corpus test, which is opt-in and skips itself when the
  # directories it walks are absent.
  doCheck = true;

  meta = with lib; {
    description = "Usefully organise directories and point out interesting files";
    homepage = "https://web.archive.org/web/2016/http://practicalthought.com/sl/";
    license = licenses.gpl3Plus;
    maintainers = [ ];
    platforms = platforms.unix;
    mainProgram = "sl";
  };
}
