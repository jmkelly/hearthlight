{ self }:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.hearthlight;
  system = pkgs.stdenv.hostPlatform.system;

  args = [
    "-listen"
    "${cfg.listenAddress}:${toString cfg.port}"
    "-data"
    "/var/lib/hearthlight/state.json"
    "-interval"
    "${toString cfg.interval}s"
    "-timeout"
    "${toString cfg.timeoutMs}ms"
    "-concurrency"
    (toString cfg.concurrency)
  ]
  ++ lib.optional (cfg.cidrs != [ ]) [
    "-cidr"
    (lib.concatStringsSep "," cfg.cidrs)
  ]
  ++ lib.optional (cfg.extraPorts != [ ]) [
    "-ports"
    (lib.concatMapStringsSep "," toString cfg.extraPorts)
  ]
  ++ lib.optional (cfg.allowedHosts != [ ]) [
    "-hosts"
    (lib.concatStringsSep "," cfg.allowedHosts)
  ]
  ++ [
    "-tailscale=${lib.boolToString cfg.tailnet}"
  ];
in
{
  options.services.hearthlight = {
    enable = lib.mkEnableOption "Hearthlight home network map";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${system}.default;
      description = "The Hearthlight package to run.";
    };

    listenAddress = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1";
      description = "Address the HTTP server binds to. Loopback by default; set 0.0.0.0 to expose it on the LAN.";
    };

    allowedHosts = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "hearthlight.lan" ];
      description = "Extra hostnames accepted in the HTTP Host header in addition to loopback names and IP literals.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8090;
      description = "TCP port for the web UI and API.";
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Open the Hearthlight port in the NixOS firewall.";
    };

    interval = lib.mkOption {
      type = lib.types.ints.positive;
      default = 60;
      description = "Seconds between network scans.";
    };

    timeoutMs = lib.mkOption {
      type = lib.types.ints.between 50 5000;
      default = 350;
      description = "Per-connection TCP timeout in milliseconds.";
    };

    concurrency = lib.mkOption {
      type = lib.types.ints.between 1 128;
      default = 16;
      description = "Number of hosts probed in parallel.";
    };

    cidrs = lib.mkOption {
      type = lib.types.listOf lib.types.str;
      default = [ ];
      example = [ "192.168.1.0/24" ];
      description = "Optional IPv4 CIDRs to scan instead of auto-detecting LAN interfaces.";
    };

    extraPorts = lib.mkOption {
      type = lib.types.listOf lib.types.port;
      default = [ ];
      example = [
        8000
        9000
      ];
      description = "Extra TCP ports to probe in addition to the built-in list.";
    };

    tailnet = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = ''
        Discover Tailscale peers by running `tailscale status --json` and probe
        them over their tailnet addresses. Requires tailscaled on the same host;
        a missing binary or a stopped daemon is logged and skipped.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.hearthlight = {
      description = "Hearthlight home network map";
      wantedBy = [ "multi-user.target" ];
      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];
      # Makes `tailscale` reachable without adding privileges. The CLI only
      # talks to the local tailscaled socket.
      path = lib.optionals cfg.tailnet [ pkgs.tailscale ];
      serviceConfig = {
        Type = "simple";
        DynamicUser = true;
        StateDirectory = "hearthlight";
        ExecStart = "${lib.getExe cfg.package} ${lib.escapeShellArgs args}";
        Restart = "on-failure";
        RestartSec = 5;
        UMask = "0077";
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
          # net.Interfaces() uses netlink on Linux.
          "AF_NETLINK"
        ];
        RestrictNamespaces = true;
        RestrictSUIDSGID = true;
        LockPersonality = true;
        CapabilityBoundingSet = "";
        AmbientCapabilities = [ ];
        SystemCallArchitectures = "native";
      };
    };

    networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [ cfg.port ];
  };
}
