{
  description = "Grammar-Aware Go Formatter";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = self.shortRev or "dirty";
      in
      {
        packages.default = pkgs.buildGo125Module {
          inherit version;

          pname = "iku";
          src = pkgs.lib.cleanSource ./.;
          vendorHash = null;
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          meta = with pkgs.lib; {
            description = "Grammar-Aware Go Formatter";
            homepage = "https://github.com/Fuwn/iku";
            license = [
              licenses.mit
              licenses.asl20
            ];
            platforms = platforms.unix;
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go_1_25 ];
        };
      }
    );
}
