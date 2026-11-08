# Hearthlight

Hearthlight is a small Go web app that draws your home network as a live
constellation. Each device is a bubble. Each TCP service orbits its device. Zoom
out for the whole map, zoom in for service names, ports and device details.

![Hearthlight demo map](docs/hearthlight.png)

Devices seen in earlier scans stay on the map as dim, dashed bubbles. Devices
that are currently reachable pulse. Every device bubble carries a service count
at overview zoom. Service bubbles and labels appear as you zoom in.

## What you get

- One static Go binary with the browser app embedded.
- Unprivileged LAN discovery using ARP plus a TCP RST liveness sweep.
- TCP service probing with HTTP title, SSH banner and TLS certificate
  fingerprints.
- Device names from reverse DNS, vendors from a curated MAC OUI table, and
  kind classification for routers, servers, computers, phones, NAS boxes,
  media players, printers and IoT devices.
- Persistent history in a small JSON state file.
- Live updates over Server-Sent Events.
- A Nix flake and a NixOS module for the server server.

## Quick start

Requirements: Linux and Go 1.23 or newer. The scanner reads `/proc/net/arp`,
so macOS and Windows are not supported.

```bash
cd hearthlight
make demo
# Open http://127.0.0.1:8090
```

Demo mode serves a fixed sample network. To scan the real LAN:

```bash
make run
# Open http://127.0.0.1:8090
```

The default interval is 60 seconds. A scan normally finishes in a few seconds on
a `/24`. Hearthlight needs no root privileges.

Build a binary:

```bash
make build
./bin/hearthlight -listen :8090
```

## Run it on the server NixOS host

Run Hearthlight natively, not in a container. The scanner needs to resolve ARP and
open TCP connections on the home LAN. The NixOS service runs as a dynamic
unprivileged user and still has normal network access.

This repository exposes:

- `packages.<system>.hearthlight` and `packages.<system>.default`
- `nixosModules.default`
- a development shell with Go and gopls

### Wire it into the existing NixOS flake

The server configuration lives in `/etc/nixos`. Add Hearthlight as
a flake input that points at this repository:

```nix
# nixos-config/flake.nix
inputs = {
  # ...
  hearthlight.url = "path:/path/to/hearthlight";
};
```

Add `hearthlight` to the flake output arguments:

```nix
outputs = { self, nixpkgs, disko, hearthlight, ... }:
```

Then import the module into the server host and enable the service:

```nix
# nixos-config/hosts/server/default.nix
imports = [
  # ...
  hearthlight.nixosModules.default
];

services.hearthlight = {
  enable = true;
  port = 8090;
  interval = 60;
};
```

This integration is already applied in `/etc/nixos` on this
machine. Its full server system closure builds successfully with:

```bash
nix build .#nixosConfigurations.server.config.system.build.toplevel
```

Deploy as usual from the NixOS checkout when the server host is ready:

```bash
cd /etc/nixos
scripts/deploy.sh
```

The module opens port 8090 in the firewall by default. From the LAN, open:

```text
http://192.0.2.30:8090
```

From the tailnet, use the server MagicDNS name:

```text
http://server:8090
```

Check the service:

```bash
ssh user@192.0.2.30
systemctl status hearthlight
journalctl -u hearthlight -f
```

The service stores its inventory at `/var/lib/hearthlight/state.json` with mode
`0600`. Delete that file to forget all history.

### Module options

| Option | Default | Meaning |
| --- | --- | --- |
| `enable` | `false` | Start the Hearthlight service. |
| `package` | flake package | Hearthlight package to run. |
| `listenAddress` | `0.0.0.0` | HTTP bind address. |
| `port` | `8090` | HTTP port. |
| `openFirewall` | `true` | Open the HTTP port in the NixOS firewall. |
| `interval` | `60` | Seconds between scans. |
| `timeoutMs` | `350` | Per-connection probe timeout. |
| `concurrency` | `16` | Hosts probed in parallel. |
| `cidrs` | `[]` | Explicit IPv4 ranges. Empty means auto-detect. |
| `extraPorts` | `[]` | Extra TCP ports to probe. |

### Standalone package check

```bash
nix build .#hearthlight
./result/bin/hearthlight -demo -listen 127.0.0.1:8090
```

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-listen` | `:8090` | HTTP listen address. |
| `-data` | `hearthlight-state.json` | State file. Empty means in-memory. |
| `-interval` | `60s` | Time between scans. Minimum 5s. |
| `-cidr` | auto | Comma-separated IPv4 CIDRs to scan. |
| `-ports` | built-in list | Extra TCP ports, for example `8000,9000-9010`. |
| `-timeout` | `350ms` | Per-connection timeout. |
| `-concurrency` | `16` | Parallel host probes. |
| `-demo` | `false` | Serve sample data without scanning. |
| `-version` | | Print the version and exit. |

Environment variables `HEARTHLIGHT_LISTEN` and `HEARTHLIGHT_DATA` set the listen address
and state path when the corresponding flag is not provided.

## How discovery works

One scan does the following:

1. Enumerates the IPv4 LAN interfaces. Networks larger than `/24` are capped to
   the local `/24`. Docker, Podman, Tailscale and other virtual interfaces are
   skipped.
2. Sends one UDP packet to each address to wake the kernel's ARP cache.
3. Connects to a closed TCP port on every address. A reset means the host is
   alive, even when it exposes no services.
4. Reads `/proc/net/arp` for MAC addresses and additional candidates.
5. Probes a curated TCP port list on each candidate.
6. Fingerprints open web ports with an HTTP title, SSH with its banner, and TLS
   ports with the certificate name.
7. Resolves PTR names and maps known MAC prefixes to vendors.
8. Merges the observation into the state file. Everything not seen recently
   becomes history rather than being deleted.

The browser gets a full snapshot over `/api/state` and receives live snapshots
over `/api/events`.

## Limits

- TCP only. UDP services such as mDNS, SSDP, DHCP and WireGuard are not
  discovered.
- A device that answers neither ARP nor TCP can be missed.
- Sleeping Wi-Fi devices can move to history while asleep. They return on the
  next scan that sees them.
- The OUI table is a curated list, not the full IEEE registry. Unknown vendors
  are shown as blank.
- Service names come from common port conventions and fingerprinting. A
  non-standard service on a well-known port can be mislabeled.
- History is kept indefinitely. Remove the state file to reset it.

## Security

Hearthlight has no authentication. Its API exposes hostnames, IP addresses, MAC
addresses and open ports. Run it only on a trusted LAN or tailnet.

The NixOS module binds to all interfaces and opens the port by default. Set
`openFirewall = false` and use an SSH tunnel or a tailnet-only firewall rule if
you do not want LAN access. Do not publish it directly to the internet.

Active service probing can trigger intrusion detection or logging on managed
networks. Use it only on networks you own or administer.

## Development

```bash
make demo     # sample data UI
make test     # unit tests
make check    # formatting, vet and tests
make build    # bin/hearthlight
```

The main packages are:

- `internal/discovery`: network enumeration, liveness, port probes and
  fingerprints.
- `internal/store`: merges observations and derives online/history states.
- `internal/api`: JSON API, SSE stream and the embedded UI.
- `internal/demo`: sample network for UI work.
- `web`: canvas map, interaction and detail panel.
