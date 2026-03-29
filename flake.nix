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
        packages.openpost = pkgs.stdenv.mkDerivation {
          pname = "openpost";
          version = "0.1.0";

          src = self;

          nativeBuildInputs = with pkgs; [
            pkgs.go_1_25
            pkgs.bun
            pkgs.cacert
          ];

          buildInputs = with pkgs; [ pkgs.sqlite ];

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
            # Copy this to .env and fill in your values

            # Required: Generate with: openssl rand -base64 32
            JWT_SECRET=your-jwt-secret-min-32-chars
            ENCRYPTION_KEY=your-encryption-key-32-chars

            # Optional
            OPENPOST_PORT=8080
            OPENPOST_DB_PATH=file:openpost.db?cache=shared&mode=rwc
            OPENPOST_FRONTEND_URL=http://localhost:8080

            # Twitter/X OAuth
            # TWITTER_CLIENT_ID=your-twitter-client-id
            # TWITTER_CLIENT_SECRET=your-twitter-client-secret

            # LinkedIn OAuth
            # LINKEDIN_CLIENT_ID=your-linkedin-client-id
            # LINKEDIN_CLIENT_SECRET=your-linkedin-client-secret

            # Threads/Meta OAuth
            # THREADS_CLIENT_ID=your-meta-app-id
            # THREADS_CLIENT_SECRET=your-meta-app-secret

            # Mastodon
            # MASTODON_REDIRECT_URI=urn:ietf:wg:oauth:2.0:oob
            # MASTODON_SERVERS='[{"name":"Instance","client_id":"","client_secret":"","instance_url":"https://mastodon.social"}]'
            ENVFILE

                        runHook postInstall
          '';

          meta = with pkgs.lib; {
            description = "OpenPost - Self-hosted social media scheduler";
            homepage = "https://github.com/rodrgds/openpost";
            license = pkgs.lib.licenses.mit;
            platforms = pkgs.lib.platforms.linux ++ pkgs.lib.platforms.darwin;
          };
        };

        packages.default = self.packages.${system}.openpost;
      }
    )
    // {
      nixosModules.default = import ./nix/module.nix;
      nixosModules.openpost = import ./nix/module.nix;
    };
}
