# OpenPost - Self-hosted social media scheduler
# Creates a single Go binary with embedded SvelteKit frontend
{
  config,
  lib,
  ...
}:
let
  cfg = config.vps.openpost;
  openpostPort = 8180;
in
{
  options.vps.openpost = {
    enable = lib.mkEnableOption "OpenPost social media scheduler";

    domain = lib.mkOption {
      type = lib.types.str;
      default = "openpost.rgo.pt";
      description = "Domain for OpenPost";
    };

    dataDir = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/openpost";
      description = "Directory for persistent data";
    };
  };

  config = lib.mkIf cfg.enable {
    # Create persistent directories
    systemd.tmpfiles.rules = [
      "d ${cfg.dataDir} 0750 root root -"
      "d ${cfg.dataDir}/db 0700 1000 1000 -"
      "d ${cfg.dataDir}/media 0700 1000 1000 -"
    ];

    # OpenPost container
    virtualisation.oci-containers.containers.openpost = {
      image = "ghcr.io/rodrgds/openpost:latest";

      environment = {
        OPENPOST_PORT = "8080";
        OPENPOST_DB_PATH = "/data/db/openpost.db";
        OPENPOST_MEDIA_PATH = "/data/media";
      };

      environmentFiles = [
        config.sops.templates.openpost-env.path
      ];

      volumes = [
        "${cfg.dataDir}/db:/data/db"
        "${cfg.dataDir}/media:/data/media"
      ];

      ports = [
        "127.0.0.1:${toString openpostPort}:8080"
      ];

      extraOptions = [
        "--network=podman"
        "--health-cmd=wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1"
        "--health-interval=30s"
        "--health-timeout=3s"
        "--health-retries=3"
      ];
    };

    # Secrets template
    sops.templates.openpost-env = {
      content = ''
        JWT_SECRET=${config.sops.placeholder.openpost_jwt_secret}
        ENCRYPTION_KEY=${config.sops.placeholder.openpost_encryption_key}
      ''
      + lib.optionalString (config.sops.placeholder ? openpost_twitter_client_id) ''
        TWITTER_CLIENT_ID=${config.sops.placeholder.openpost_twitter_client_id}
        TWITTER_CLIENT_SECRET=${config.sops.placeholder.openpost_twitter_client_secret}
      ''
      + lib.optionalString (config.sops.placeholder ? openpost_linkedin_client_id) ''
        LINKEDIN_CLIENT_ID=${config.sops.placeholder.openpost_linkedin_client_id}
        LINKEDIN_CLIENT_SECRET=${config.sops.placeholder.openpost_linkedin_client_secret}
      ''
      + lib.optionalString (config.sops.placeholder ? openpost_threads_client_id) ''
        THREADS_CLIENT_ID=${config.sops.placeholder.openpost_threads_client_id}
        THREADS_CLIENT_SECRET=${config.sops.placeholder.openpost_threads_client_secret}
      ''
      + lib.optionalString (config.sops.placeholder ? openpost_mastodon_servers) ''
        MASTODON_SERVERS=${config.sops.placeholder.openpost_mastodon_servers}
      '';
      mode = "0400";
    };

    # Caddy reverse proxy integration
    vps.caddy.internalPorts.openpost = openpostPort;

    # Caddy virtual host (handled by dynamicVirtualHosts in caddy module)
    # Or can be customized:
    services.caddy.virtualHosts."${cfg.domain}" = {
      extraConfig = ''
        reverse_proxy localhost:${toString openpostPort}

        header {
          Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
          X-Content-Type-Options "nosniff"
          X-Frame-Options "SAMEORIGIN"
          X-XSS-Protection "1; mode=block"
          Referrer-Policy "strict-origin-when-cross-origin"
        }
      '';
    };
  };
}
