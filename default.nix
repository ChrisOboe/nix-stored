{
  buildGoModule,
  lib,
  oapi-codegen,
  pkgs,
  stdenv,
}:
buildGoModule rec {
  pname = "nix-stored";
  version = "1.0.0";

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
