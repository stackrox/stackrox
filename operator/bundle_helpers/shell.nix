let
  nixpkgsPinned = import (builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/nixos-24.11.tar.gz";
  }) {};
  pkgs = nixpkgsPinned.pkgs;
in
pkgs.mkShell {
  buildInputs = [
    nixpkgsPinned.python39
    nixpkgsPinned.python39Packages.pip-tools
  ];
}
