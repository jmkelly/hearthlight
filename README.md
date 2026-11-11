# Hearthlight

Hearthlight is a small Go web app that draws your home network as a live map.
Each device is a bubble. Each TCP service orbits its device. Zoom out for the
whole map, zoom in for service glyphs, names, ports and device details.

Hearthlight also maps your Tailscale tailnet as a second territory. Peers come
from `tailscale status --json` and are probed over their tailnet addresses, then
merged with LAN devices. Hosts that are on both networks sit in the bright
overlap lens.

![Hearthlight demo map](docs/hearthlight.png)

Devices seen in earlier scans stay on the map as dim, dashed bubbles. Devices
that are currently reachable pulse. Every device bubble carries a service count
at overview zoom. Service bubbles and labels appear as you zoom in.

## What you get

- One static Go binary with the browser app embedded.
- Unprivileged LAN discovery using ARP plus a TCP RST liveness sweep.
- The current Wi-Fi network name (SSID), when the host is wireless.
- Tailnet discovery from the local Tailscale daemon, merged with LAN devices by
  hostname and self identity.
- TCP service probing with HTTP title, SSH banner and TLS certificate
  fingerprints, on both networks.
- A dual-membership map: warm LAN and cool tailnet territories drawn as soft,
  lasso-like clouds, and a lens where hosts belong to both.
- Device names from reverse DNS, vendors from a curated MAC OUI table, and
  kind classification for routers, servers, computers, phones, NAS boxes,
  media players, printers and IoT devices.
- Persistent history in a small JSON state file.
- Live updates over Server-Sent Events.
- Live traffic edges between this host and the devices it talks to.
- A Nix flake and a NixOS module.
- A theme catalog with dark, light and auto modes, stored per browser.

## Themes

Hearthlight Dark stays the default. Open **Settings → Appearance** (the gear
button, or press `s`) to pick a theme family and mode. The choice is stored in
this browser's `localStorage` under `hearthlight.theme.v1`, not on the server,
so different browsers and devices can use different themes.

Dark and Light pin a mode. Auto follows the operating system's
`prefers-color-scheme` and updates when it changes. Families without a light
variant keep their available palette and Settings says so.

Browse the [theme gallery and authoring guide](docs/themes.md), including the
[Omarchy integration notes](docs/themes.md#omarchy). The Omarchy plugin and its
stock mapping are not part of this build yet.

For screenshots or embedding, set the variant with a URL parameter. The override
applies to that page load and does not overwrite the stored choice:

```text
http://127.0.0.1:8090/?theme=hearthlight-light
http://127.0.0.1:8090/?theme=kanagawa-wave&mode=auto
http://127.0.0.1:8090/?mode=light
```

`theme` takes a variant ID from the gallery. `mode` takes `dark`, `light` or
`auto` and keeps the current family.

## Quick start

Requirements: Go 1.23 or newer. Real network scanning works on Linux, macOS
and Windows. Hearthlight reads the native ARP table and probes TCP services
without root privileges. On macOS, allow your terminal or the Hearthlight
binary local network access when prompted.

```bash
git clone https://github.com/jmkelly/hearthlight.git
cd hearthlight
make demo
# Open http://127.0.0.1:8090
```

Demo mode serves a fixed sample network, including a simulated tailnet. To scan
the real LAN:

```bash
make run
# Open http://127.0.0.1:8090
```

If Tailscale is installed and `tailscaled` is running, tailnet peers are picked
up automatically. Pass `-tailscale=false` to keep Hearthlight LAN-only.

The default interval is 60 seconds. A scan normally finishes in a few seconds on
a `/24`. Hearthlight needs no root privileges.

Build a binary:

```bash
make build
./bin/hearthlight -listen 127.0.0.1:8090
```

## Prebuilt binaries

Versioned Linux, macOS and Windows archives are published on the
[releases page](https://github.com/jmkelly/hearthlight/releases). All builds
support real scanning; add `-demo` to serve the sample network instead.

## Run it on a NixOS host

Run Hearthlight natively, not in a container. The scanner needs to resolve ARP
and open TCP connections on the home LAN. The NixOS service runs as a dynamic
unprivileged user and still has normal network access.

This repository exposes:

- `packages.<system>.hearthlight` and `packages.<system>.default`
- `nixosModules.default`
- a development shell with Go and gopls

### Add it to a NixOS flake

Add Hearthlight as a flake input:

```nix
# flake.nix
inputs = {
  # ...
  hearthlight.url = "github:jmkelly/hearthlight";
};
```

Add `hearthlight` to the flake output arguments:

```nix
outputs = { self, nixpkgs, hearthlight, ... }:
```

Import the module and enable the service:

```nix
# configuration.nix
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

The module binds to loopback and leaves the firewall closed by default. To
expose it on the LAN, set `listenAddress = "0.0.0.0"`, add any DNS names you
use to `allowedHosts`, and set `openFirewall = true`. Put authentication in
front of it with a reverse proxy or run the binary directly with `-token`.
From another machine, open:

```text
http://<host>:8090
```

Tailnet discovery is on by default. When `tailnet = true` the module puts
`pkgs.tailscale` on the service `PATH` so Hearthlight can run
`tailscale status --json`; it adds no privileges. Hearthlight talks to the local
`tailscaled` socket, which is normally accessible to every local user
(`srw-rw-rw-`). If you have hardened that socket, grant the dynamic
`hearthlight` user access separately. tailscaled itself must already be running
and logged in:

```nix
services.tailscale.enable = true;
services.hearthlight.tailnet = true;
```

Check the service:

```bash
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
| `listenAddress` | `127.0.0.1` | HTTP bind address. |
| `allowedHosts` | `[]` | Extra hostnames accepted in the HTTP `Host` header. |
| `port` | `8090` | HTTP port. |
| `openFirewall` | `false` | Open the HTTP port in the NixOS firewall. |
| `interval` | `60` | Seconds between scans. |
| `timeoutMs` | `350` | Per-connection probe timeout. |
| `concurrency` | `16` | Hosts probed in parallel. |
| `cidrs` | `[]` | Explicit IPv4 ranges. Empty means auto-detect. |
| `extraPorts` | `[]` | Extra TCP ports to probe. |
| `tailnet` | `true` | Discover Tailscale peers and put `tailscale` on the service `PATH`. |

### Standalone package check

```bash
nix build .#hearthlight
./result/bin/hearthlight -demo -listen 127.0.0.1:8090
```

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-listen` | `127.0.0.1:8090` | HTTP listen address. |
| `-data` | `hearthlight-state.json` | State file. Empty means in-memory. |
| `-interval` | `60s` | Time between scans. Minimum 5s. |
| `-cidr` | auto | Comma-separated IPv4 CIDRs to scan. |
| `-ports` | built-in list | Extra TCP ports, for example `8000,9000-9010`. |
| `-timeout` | `350ms` | Per-connection timeout. |
| `-concurrency` | `16` | Parallel host probes. |
| `-demo` | `false` | Serve sample data without scanning. |
| `-tailscale` | `true` | Discover Tailscale peers via `tailscale status --json`. |
| `-traffic` | `true` | Sample live traffic between this host and mapped devices every 2s. |
| `-hosts` | | Extra hostnames accepted in the HTTP `Host` header, comma-separated. |
| `-token` | | Shared token required for the API. Empty disables it. |
| `-version` | | Print the version and exit. |

Environment variables `HEARTHLIGHT_LISTEN`, `HEARTHLIGHT_DATA`,
`HEARTHLIGHT_HOSTS` and `HEARTHLIGHT_TOKEN` set the listen address, state path,
allowed hostnames and API token when the corresponding flag is not provided.
`HEARTHLIGHT_TAILSCALE` overrides the path or name of the `tailscale` binary.

## How discovery works

One scan does the following:

1. Enumerates the IPv4 LAN interfaces. Networks larger than `/24` are capped to
   the local `/24`. Docker, Podman, Tailscale and other virtual interfaces are
   skipped; the tailnet is discovered separately.
2. Sends one UDP packet to each address to wake the kernel's ARP cache.
3. Connects to a closed TCP port on every address. A reset means the host is
   alive, even when it exposes no services.
4. Reads the native ARP table for MAC addresses and additional candidates:
   `/proc/net/arp` on Linux, `arp -an` on macOS and `arp -a` on Windows.
5. Probes a curated TCP port list on each candidate.
6. Fingerprints open web ports with an HTTP title, SSH with its banner, and TLS
   ports with the certificate name.
7. Resolves PTR names and maps known MAC prefixes to vendors.
8. Merges the observation into the state file. Everything not seen recently
   becomes history rather than being deleted.
9. Unless `-tailscale=false`, runs `tailscale status --json` with a 5 second
   timeout and reads Self plus every Peer: node ID/public key, hostname,
   MagicDNS name, OS, IPv4 address in `100.64.0.0/10`, tags, online state and
   last seen.
10. Probes online peers over the same port list, timeout and concurrency as the
    LAN, then fingerprints their services the same way.
11. Matches peers to LAN devices by unique normalized short hostname
    (case-insensitive, trailing dot and DNS suffix removed) and matches the
    local host through the `Self` flag. Ambiguous duplicate hostnames are never
    merged.
12. Merges memberships. A dual host keeps its LAN identity, MAC and history;
    services merge by port and record both reach paths. A tailnet-only peer gets
    the stable ID `ts:<node ID or public key>`. If an online peer refuses all TCP
    probes it stays a device because Tailscale says it is up.

A missing `tailscale` binary, a stopped `tailscaled`, a timeout or malformed
JSON is logged and skipped. That reading is ignored, the LAN scan still
succeeds, and the previous tailnet state is kept.

Each scan also reads the current Wi-Fi SSID, with `iwgetid` or `nmcli` on
Linux, `networksetup` or `ipconfig` on macOS, and `netsh` on Windows. A wired
host shows no SSID. When the tool is missing or the OS withholds the name
(macOS may redact it without location access), the reading is skipped and the
scan continues. The SSID appears in the header and on the LAN territory label.
It is live host state and is never written to the state file.

The browser gets a full snapshot over `/api/state` and receives live snapshots
over `/api/events`.

## Live traffic

Every 2 seconds Hearthlight samples traffic between this host and the devices
on the map, and pushes it to the browser as a `traffic` event on
`/api/events`. Nothing is persisted.

- **Tailnet edges** carry real byte rates from the per-peer WireGuard counters
  in `tailscale status --json`. Particles flow toward this host for received
  bytes and away from it for sent bytes; more and faster particles mean more
  bytes.
- **LAN edges** come from the OS TCP connection table: `/proc/net/tcp` on
  Linux, `netstat -an -p tcp` on macOS and `GetExtendedTcpTable` on Windows.
  They show established connection counts, drawn as a marching dash, because
  per-connection byte counts need admin rights on Windows.
- While a scan runs, the previous connection table is reused so probe
  connections do not show up as traffic.

Hearthlight sees only its own host's traffic. On a switched network an
unprivileged host cannot observe traffic between two other devices.

## Limits

- TCP only. UDP services such as mDNS, SSDP, DHCP and WireGuard are not
  discovered.
- A device that answers neither ARP nor TCP can be missed.
- macOS may withhold local network access until you allow it under System
  Settings → Privacy & Security → Local Network. A third-party firewall can
  also block the probes on Windows or macOS.
- Sleeping Wi-Fi devices can move to history while asleep. They return on the
  next scan that sees them.
- Tailnet discovery shells out to the local `tailscale` CLI. It needs a running,
  logged-in `tailscaled`; there is no API token or remote discovery.
- Tailnet peers are matched by normalized short hostname. If two LAN devices or
  two peers share a hostname, the peer stays separate instead of risking a wrong
  merge. A peer whose MagicDNS name differs from its reverse-DNS name will not
  merge either.
- Tailnet ACLs may block service probes. The peer still appears as an online
  device from Tailscale status, just without services.
- Tailscale `Online` is the liveness signal for tailnet members; a failed TCP
  probe never marks an online peer offline. LAN members still use recent
  observation time.
- Tags, OS and MagicDNS names are informational and come from Tailscale status.
- The OUI table is a curated list, not the full IEEE registry. Unknown vendors
  are shown as blank.
- Service names come from common port conventions and fingerprinting. A
  non-standard service on a well-known port can be mislabeled.
- History is kept indefinitely. Remove the state file to reset it.

## Security

Hearthlight defaults to a local-only shape.

- The HTTP server binds to `127.0.0.1:8090`. Change `-listen` or
  `services.hearthlight.listenAddress` only when you mean to expose it.
- Every request must carry an allowed `Host` header: loopback names, IP
  literals, or a hostname passed to `-hosts`. A DNS-rebinding page whose
  hostname points at Hearthlight is rejected.
- `POST /api/scan` requires a same-origin `Origin` or a `Sec-Fetch-Site` of
  `same-origin` or `none`, so another web page cannot queue scans.
- Set `-token` (or `HEARTHLIGHT_TOKEN`) to require a shared token on every route
  except `/healthz`. Open `http://<host>:8090/?token=<token>` once and the
  server stores an HttpOnly, SameSite=Strict session cookie. API clients can
  send `Authorization: Bearer <token>` instead. The token is compared in
  constant time.
- The API still has no user accounts. Its inventory exposes hostnames, IP
  addresses, MAC addresses and open ports. Put a reverse proxy with
  authentication and TLS in front when the network is not trusted. Do not
  publish it directly to the internet; use an SSH tunnel or a tailnet-only
  firewall rule instead.

The NixOS module leaves the firewall closed by default. Set `openFirewall =
true` only when you want LAN access.

Active service probing can trigger intrusion detection or logging on managed
networks. Use it only on networks you own or administer. The same applies to
your tailnet: probes cross Tailscale ACLs and show up as connections from this
host, so make sure your policy allows them.

State files created before tailnet support are migrated automatically: existing
devices and services default to `local=true`, `tailnet=false`. The state file
version is bumped on the next save.

## Development

```bash
make demo     # sample data UI
make test     # unit tests
make check    # formatting, vet and tests
make build    # bin/hearthlight
```

The main packages are:

- `internal/discovery`: network enumeration, liveness, port probes and
  fingerprints, plus Tailscale status discovery.
- `internal/store`: merges LAN and tailnet observations into one inventory and
  derives online/history states.
- `internal/api`: JSON API, SSE stream and the embedded UI.
- `internal/demo`: sample network for UI work, including a simulated tailnet.
- `web`: canvas map, Aurora territories, interaction and detail panel.

## Releases

Release archives are built from tags. To publish a new version, update
`VERSION`, commit, and push a matching tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs the tests, builds Linux, macOS and Windows archives,
generates `checksums.txt`, and attaches everything to the GitHub release.

## License

MIT. See [LICENSE](LICENSE).
