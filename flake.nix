{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:

    flake-utils.lib.eachSystem [ "x86_64-linux" ] (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "${builtins.substring 0 4 self.sourceInfo.lastModifiedDate}.${
          builtins.substring 4 2 self.sourceInfo.lastModifiedDate
        }.${builtins.substring 6 2 self.sourceInfo.lastModifiedDate}-${
          self.sourceInfo.shortRev or self.sourceInfo.dirtyShortRev
        }";
      in
      rec {
        packages = rec {
          default = nix-stored;
          nix-stored = pkgs.callPackage ./default.nix { version = version; };
        };

        devShell = pkgs.mkShell {
          buildInputs = [
            # dev
            pkgs.go
            pkgs.oapi-codegen
            pkgs.golangci-lint
            pkgs.govulncheck
          ];
        };
      }
    );
}
