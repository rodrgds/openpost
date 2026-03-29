# OpenPost NixOS Module
# Self-hosted social media scheduler
{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.openpost;
in
{
  options.services.openpost = {
    enable = lib.mkEnableOption "OpenPost social media scheduler";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ../. { };
      defaultText = lib.literalExpression "pkgs.callPackage ./. { }";
      description = "The OpenPost package to use.";
    };

    dataDir = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/openpost";
      description = "Directory for OpenPost data (database and media).";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "Port to run OpenPost on.";
    };

    environment = lib.mkOption {
      type = lib.types.attrsOf lib.types.str;
      default = { };
      description = ''
        Environment variables to pass to OpenPost.
        Required: JWT_SECRET, ENCRYPTION_KEY
        Optional: TWITTER_CLIENT_ID, TWITTER_CLIENT_SECRET, etc.
      '';
      example = lib.literalExpression ''
        {
          JWT_SECRET = "your-secret";
          ENCRYPTION_KEY = "your-encryption-key";
        }
      '';
    };

    environmentFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = ''
        Path to a file containing environment variables.
        Useful for secrets that shouldn't be in the Nix store.
        Format: KEY=value (one per line)
      '';
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Open the firewall for OpenPost port.";
    };
  };

  config = lib.mkIf cfg.enable {
    # Create data directory
    systemd.tmpfiles.rules = [
      "d '${cfg.dataDir}' 0750 openpost openpost -"
      "d '${cfg.dataDir}/db' 0750 openpost openpost -"
      "d '${cfg.dataDir}/media' 0750 openpost openpost -"
    ];

    # Create openpost user
    users.users.openpost = {
      isSystemUser = true;
      group = "openpost";
      home = cfg.dataDir;
      createHome = false;
    };
    users.groups.openpost = { };

    # Systemd service
    systemd.services.openpost = {
      description = "OpenPost - Social Media Scheduler";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        Type = "simple";
        User = "openpost";
        Group = "openpost";
        WorkingDirectory = cfg.dataDir;
        ExecStart = lib.getExe cfg.package;
        Restart = "on-failure";
        RestartSec = "5s";

        # Security hardening
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [ cfg.dataDir ];
        CapabilityBoundingSet = "";
        SystemCallFilter = [
          "@system-service"
          "~@privileged"
        ];
      };

      environment = cfg.environment // {
        OPENPOST_PORT = toString cfg.port;
        OPENPOST_DB_PATH = "${cfg.dataDir}/db/openpost.db";
        OPENPOST_MEDIA_PATH = "${cfg.dataDir}/media";
      };

      environmentFile = lib.mkIf (cfg.environmentFile != null) cfg.environmentFile;
    };

    # Firewall
    networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [ cfg.port ];

    # Package
    environment.systemPackages = [ cfg.package ];
  };
}
