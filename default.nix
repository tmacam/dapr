{ go ? "go_1_18" }:

let
  # get a normalized set of packages, from which
  # we will install all the needed dependencies
  pkgs = import <nixpkgs> {};
in
  pkgs.mkShell {
    buildInputs = [
      pkgs.${go}
      pkgs.protobuf
      pkgs.protoc-gen-go
      pkgs.protoc-gen-go-grpc
    ];
    shellHook = ''
      export NIX_ENV=dev
    '';
  }

