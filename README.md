# Nix StoreD

Nix StoreD (the D stands for Daemon) is a binary cache server for Nix,
designed to store and serve NAR files. It provides endpoints to upload,
download, and check the existence of NAR files. The server is built using
Go and leverages the oapi-codegen library for generating API code from
OpenAPI specifications. It includes basic authentication middleware to
secure endpoints and logging middleware for request tracing. The server
is configured via environment variables, making it flexible and easy to
deploy in various environments.

## Configuration

The software can be configured using the following environment variables:

- `NIX_STORED_PATH`:             The path where NAR files are stored. Default
-                                is `/var/lib/nixStored`.
- `NIX_STORED_LISTEN_INTERFACE`: The interface and port on which the server
-                                listens. Default is `127.0.0.1:8100`.
- `NIX_STORED_USER_READ`:        The username for read access. Default is empty.
- `NIX_STORED_USER_READ_PASS`:   The password for read access. Default is empty.
- `NIX_STORED_USER_WRITE`:       The username for write access. Default is empty.
- `NIX_STORED_USER_WRITE_PASS`:  The password for write access. Default is empty.

Set these environment variables in your deployment environment to
customize the server's behavior. The store path from Nix Stored is completely
independend from your Nix Store.

# USP (Unique Selling Point)
- It allows uploading via http (at least the original nix-serve doesn't)
  and there is almost no documentation about this (but nix supports this)
  So you can do all the nice stuff you can to with HTTP (e.g. using nice
  reverse proxies). 
- It's written in Go so deployment is easy and performance should be very
  good while preventing potential security issues servers with manually
  managed memory can have. Also since there are very few
- It's just a few lines of code. It just hosts nix stuff. It doesn't sign
  packages (which is imho wierd anyways for a server that supports
  uploading since i want the builder to sign the stuff, not the server).

# Usecases
## CI Cache
Modern CI systems are somewhat stateless. To prevent rebuilding everything
again and again a cache can be used. This software can be your cache since
it integrates nicely to nix.

## Sharing Binaries
I use NixOS on lots of different devices (Mediacenters, a 3D-Printer,
a Gaming PC, a Desktop PC, a Notebook and even an Arcade Machine). I
share as much of the config as possible. Often some very specific
configuration is needed so stuff needs to be built locally since it's
not in the nixpkgs cache. To prevent that multiple devices build the
same things i need a place to centrally store the builds so if device
A has already built the software, device B can use it.

# Setup
## Installation
We use flakes so you can run it direcly via ```nix run github:ChrisOboe/nix-stored```
If you want it in your NixOS i'd recommend to add this to your flake as input
and add the nix-stored package from this flake to your nixpkgs overlay.

## Daemon
At first you want to get Nix StoreD running. Here's how you set it up on NixOS:
```
systemd.services.nix-stored = {
  serviceConfig.ExecStart = "${pkgs.nix-stored}/bin/nix-stored";
  environmentVariables = {
    NIX_STORED_LISTEN_INTERFACE="0.0.0.0:8100";
    # or any other settings
  };
  after = ["network.target"];
  wantedBy = ["multi-user.target"];
};
```

## Nix Builder
Now you want to get your system the builds stuff via nix to upload it to
nix-stored. You can configure this as
[post-build-hook](https://nix.dev/guides/recipes/post-build-hook.html)

## Nix Consumer
Just add your nix-stored as nix substituter. Just make sure the consumer knows
the public key(s) of the builder(s).

# TODO
- I'd propably add a nixos module config to this repo, so you can directly use
  the module instead of creating the systemd.service manually.
- Maybe do some performance benchmarks to see how this implementation compares
  to the other ones.

# Related Software
- A S3 Compatible server (i experimented with S3 before writing this, but it
  had severe performance problems. So sever it made any operation basically
  unusable). So i'm not sure how compatible nix is with the different S3
  servers arround.
- The original [nix-serve](https://github.com/edolstra/nix-serve) perl script
- [nix-serve-ng](https://github.com/aristanetworks/nix-serve-ng) written in haskell
- [eris](https://github.com/thoughtpolice/eris) written in perl
- [nix-cache](https://github.com/serokell/nix-cache) which seems dead
- [harmonia](https://github.com/nix-community/harmonia) written in rust. TBH i
  just found out about harmonia after writing this. So maybe that could be
  interesting.
