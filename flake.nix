{
  description = "sl - usefully organise directories and point out interesting files";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      # The systems the program is known to build on; `platforms.unix` in the
      # derivation is wider than this list, deliberately: these are the ones
      # somebody can install here without cross-compiling by hand.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "i686-linux"
        "powerpc64le-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        sl-go = pkgs.callPackage ./packaging/nix/default.nix { };
        default = self.packages.${pkgs.system}.sl-go;
      });

      # `nix run` works because the derivation sets meta.mainProgram = "sl".
      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.system}.sl-go}/bin/sl";
        };
      });

      # `nixpkgs.overlays = [ sl.overlays.default ];` then `pkgs.sl-go`.
      overlays.default = final: _prev: {
        sl-go = final.callPackage ./packaging/nix/default.nix { };
      };

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gnumake
          ];
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt-rfc-style);
    };
}
