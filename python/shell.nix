{ pkgs ? import <nixpkgs> {} }:
let
in
pkgs.mkShell {
  packages = [ pkgs.ripgrep pkgs.go_1_22 pkgs.python3 pkgs.python3Packages.flask pkgs.python3Packages.numpy pkgs.ffmpeg-full pkgs.python3Packages.pillow pkgs.python3Packages.aiohttp pkgs.python3Packages.aiofiles ];
  nativeBuildInputs = [ pkgs.pkg-config pkgs.bzip2 pkgs.zlib pkgs.iconv ] ++ pkgs.lib.optionals pkgs.stdenv.isDarwin [
    pkgs.darwin.apple_sdk.frameworks.VideoToolbox
    pkgs.darwin.apple_sdk.frameworks.OpenGL
    pkgs.darwin.apple_sdk.frameworks.AppKit
  ];
  shellHooks = ''
    export PKG_CONFIG_PATH=$HOME/livepeer/compiled/lib/pkgconfig
  '';
}
