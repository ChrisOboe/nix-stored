{
  buildGoModule,
  lib,
  oapi-codegen,
  pkgs,
  stdenv,
  version,
}:
buildGoModule rec {
  pname = "nix-stored";
  inherit version;

  src = ./src;
  vendorHash = null;

  nativeBuildInputs = [ oapi-codegen ];

  buildPhase = ''
    mkdir api
    go generate
    go build -ldflags "-s -w" nix-stored.go
  '';

  installPhase = ''
    mkdir -p $out/bin
    cp nix-stored $out/bin/
  '';

  meta = {
    description = "a nix store daemon";
    homepage = "https://github.com/ChrisOboe/nix-stored";
    maintainers = [ "chris@ruckstetter.com" ];
    platforms = lib.platforms.linux;
  };
}
