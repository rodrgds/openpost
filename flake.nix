{
  description = "OpenPost - A lightweight, self-hosted social media scheduler";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        # Build from source - requires --impure flag due to bun network access
        packages.openpost-from-source = pkgs.stdenv.mkDerivation {
          pname = "openpost";
          version = "0.1.0";

          src = self;

          nativeBuildInputs = with pkgs; [
            go_1_25
            bun
            cacert
          ];

          buildInputs = with pkgs; [ sqlite ];

          buildPhase = ''
            export CGO_ENABLED=1
            export HOME=$TMPDIR
            runHook preBuild

            cd web
            bun install --backend=online
            bun run build
            cd ..

            cd backend
            go build -o openpost ./cmd/openpost
            cd ..

            runHook postBuild
          '';

          installPhase = ''
                        runHook preInstall

                        mkdir -p $out/bin
                        cp backend/openpost $out/bin/

                        mkdir -p $out/share/openpost
                        cat > $out/share/openpost/.env.template << 'ENVFILE'
            # OpenPost Environment Configuration
            JWT_SECRET=your-jwt-secret
            ENCRYPTION_KEY=your-encryption-key
            OPENPOST_PORT=8080
            ENVFILE

                        runHook postInstall
          '';

          meta = with pkgs.lib; {
            description = "OpenPost - Self-hosted social media scheduler (built from source)";
            homepage = "https://github.com/rodrgds/openpost";
            license = pkgs.lib.licenses.mit;
            platforms = pkgs.lib.platforms.linux ++ pkgs.lib.platforms.darwin;
          };
        };

        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_25
            bun
            sqlite
            git
          ];

          shellHook = ''
            echo "OpenPost Development Shell"
            echo "Build locally with: cd web && bun install && bun run build && cd ../backend && go build -o openpost ./cmd/openpost"
          '';
        };

        # Default package - builds from source
        packages.default = self.packages.${system}.openpost-from-source;
      }
    )
    // {
      nixosModules.default = import ./nix/module.nix;
      nixosModules.openpost = import ./nix/module.nix;
    };
}
