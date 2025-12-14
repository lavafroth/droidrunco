{
  description = "flake for github:lavafroth/droidrunco";

  outputs =
    {
      nixpkgs,
      ...
    }:
    let
      forAllSystems =
        f:
        nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {

        default = pkgs.buildGoModule {
          pname = "droidrunco";
          version = "3.0.0";

          buildInputs = [ pkgs.makeWrapper ];

          src = ./.;
          vendorHash = "sha256-oTupqkJ4Y/NBlIv5ZbSGaIngRIneiVaOsrvzeJJ5aqk=";

          postFixup = ''
            wrapProgram $out/bin/droidrunco \
              --set PATH ${
                pkgs.lib.makeBinPath [
                  pkgs.android-tools
                ]
              }
          '';
        };
      });

      devShells = forAllSystems (pkgs: {

        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            stdenv.cc.cc
            android-tools
          ];
        };

      });

    };
}
