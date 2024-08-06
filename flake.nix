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
      in
      rec {
        packages = rec {
          default = nix-stored;
          nix-stored = pkgs.callPackage ./default.nix { };
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
