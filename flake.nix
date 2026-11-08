{
  description = "Hearthlight: a live map of the devices and services on your home network";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      version = nixpkgs.lib.removeSuffix "\n" (builtins.readFile ./VERSION);
    in
    {
      packages = forAllSystems (pkgs: rec {
        hearthlight = pkgs.buildGoModule {
          pname = "hearthlight";
          inherit version;
          src = self;
          vendorHash = null;
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];
          meta = {
            description = "A live map of the devices and services on your home network";
            homepage = "https://github.com/jmkelly/hearthlight";
            license = pkgs.lib.licenses.mit;
            mainProgram = "hearthlight";
          };
        };
        default = hearthlight;
      });

      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.default}/bin/hearthlight";
        };
      });

      nixosModules.default = import ./nix/module.nix { inherit self; };

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
          ];
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt-rfc-style);
    };
}
