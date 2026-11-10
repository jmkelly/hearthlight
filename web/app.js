(() => {
  "use strict";

  const TAU = Math.PI * 2;
  const SERVICE_R = 8.5;
  const SERVICE_GLYPH_ZOOM = 1.1;
  const DOCK_MIN_WIDTH = 1100;
  const DEVICE_SUMMARY_THROTTLE_MS = 5000;
  // Movement below this many screen pixels stays a click, not a drag.
  const DRAG_THRESHOLD = 4;
  const FONT = '"Inter", "SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif';
  const MONO = 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace';

  const KIND = {
    router: { label: "Router", color: "#ffd166" },
    server: { label: "Server", color: "#5ee7ff" },
    computer: { label: "Computer", color: "#8fb3ff" },
    phone: { label: "Phone", color: "#ff9ec7" },
    nas: { label: "NAS", color: "#5ee0a5" },
    media: { label: "Media", color: "#b18cff" },
    iot: { label: "IoT", color: "#f08fc8" },
    printer: { label: "Printer", color: "#ffb86b" },
    device: { label: "Device", color: "#94a3b8" },
  };

  const CATEGORY = {
    web: "#5ee7ff",
    media: "#b18cff",
    data: "#ffb454",
    remote: "#5ee0a5",
    infra: "#6aa9ff",
    iot: "#f08fc8",
    other: "#93a4b8",
  };

  const TRAFFIC = {
    lan: { label: "LAN", color: "#ffd166", bend: -1 },
    tailnet: { label: "Tailnet", color: "#5ee7ff", bend: 1 },
  };

  // Glyph shapes live in an -4..4 box with a 1-unit stroke, so one table
  // feeds both the panel SVG and the canvas bubble.
  const SERVICE_GLYPHS = {
    globe: [
      ["ring", 0, 0, 3],
      ["ellipse", 0, 0, 1.5, 3],
      ["path", "M -3 0 H 3"],
    ],
    lock: [
      ["path", "M -1.6 -0.1 V -1.5 A 1.6 1.6 0 0 1 1.6 -1.5 V -0.1"],
      ["rect", -2.4, -0.1, 4.8, 3.2, 0.5],
      ["dot", 0, 1.1, 0.42],
    ],
    terminal: [
      ["rect", -3, -2.3, 6, 4.6, 0.6],
      ["path", "M -1.7 -0.9 L -0.3 0 L -1.7 0.9"],
      ["path", "M 0.6 1 H 2"],
    ],
    folder: [
      ["path", "M -3.2 -2.3 H -1 L -0.1 -1.4 H 3.2 V 2.3 H -3.2 Z"],
    ],
    mail: [
      ["rect", -3.2, -2.1, 6.4, 4.2, 0.5],
      ["path", "M -3.1 -1.8 L 0 0.5 L 3.1 -1.8"],
    ],
    radio: [
      ["dot", 0, 0.5, 0.5],
      ["path", "M -1.1 1.8 Q -2.3 0.3 -1.1 -0.9"],
      ["path", "M -2 2.6 Q -3.6 0.3 -2 -1.7"],
      ["path", "M 1.1 1.8 Q 2.3 0.3 1.1 -0.9"],
      ["path", "M 2 2.6 Q 3.6 0.3 2 -1.7"],
    ],
    home: [
      ["path", "M -3.3 0.6 L 0 -2.8 L 3.3 0.6"],
      ["path", "M -2.5 -0.1 V 3.3 H 2.5 V -0.1"],
    ],
    play: [
      ["ring", 0, 0, 3],
      ["fill", "M -1 -1.7 L 2 0 L -1 1.7 Z"],
    ],
    printer: [
      ["rect", -1.6, -3, 3.2, 1.9, 0.2],
      ["rect", -3.2, -1.2, 6.4, 3, 0.6],
      ["rect", -1.8, 1.3, 3.6, 1.8, 0.2],
    ],
    dot: [
      ["dot", 0, 0, 1.5],
    ],
  };

  // Service names win over products, so an HTTPS row stays a lock even when
  // the product also serves HTTP. Unknown services fall back to the category.
  const SERVICE_GLYPH_RULES = [
    [/\bhome[\s-]*assistant\b/i, "home"],
    [/\b(mqtt|mosquitto|zigbee2mqtt)\b/i, "radio"],
    [/\b(smtp|imap|pop3|exchange|mail)\b/i, "mail"],
    [/\b(smb|nfs|afp|ftp|sftp|webdav|netbios)\b/i, "folder"],
    [/\b(ssh|telnet|rdp|vnc|shell|terminal)\b/i, "terminal"],
    [/\b(ipp|lpd|lpr|raw|print(?:er)?|laserjet|deskjet|officejet)\b/i, "printer"],
    [/\b(plex|jellyfin|emby|kodi|cast|airplay|dlna|rtsp)\b/i, "play"],
    [/\b(https|tls|ssl)\b/i, "lock"],
    [/\b(http|dns|web|websocket|proxy|nginx|caddy|apache|traefik)\b/i, "globe"],
  ];

  const CATEGORY_GLYPHS = {
    web: "globe",
    media: "play",
    data: "folder",
    remote: "terminal",
    infra: "globe",
    iot: "radio",
    other: "dot",
  };

  const clamp = (value, low, high) => Math.min(high, Math.max(low, value));
  const lerp = (a, b, t) => a + (b - a) * t;
  const smoothstep = (edge0, edge1, value) => {
    const t = clamp((value - edge0) / (edge1 - edge0), 0, 1);
    return t * t * (3 - 2 * t);
  };
  const hashString = (value) => {
    let hash = 2166136261;
    for (let i = 0; i < value.length; i += 1) {
      hash ^= value.charCodeAt(i);
      hash = Math.imul(hash, 16777619);
    }
    return hash >>> 0;
  };
  const truncate = (value, max) => {
    const text = String(value);
    return text.length > max ? `${text.slice(0, max - 1)}…` : text;
  };
  const escapeHTML = (value) =>
    String(value).replace(/[&<>"']/g, (character) => {
      switch (character) {
        case "&": return "&amp;";
        case "<": return "&lt;";
        case ">": return "&gt;";
        case '"': return "&quot;";
        default: return "&#39;";
      }
    });

  // Only local suffixes are stripped for display; public domains stay intact.
  // Store matching and merge keys keep using the full hostname.
  const LOCAL_HOST_SUFFIXES = [".home.arpa", ".internal", ".local", ".lan", ".home"];
  const displayName = (device) => {
    let name = String(device.hostname || "").trim().replace(/\.$/, "");
    const lower = name.toLowerCase();
    for (const suffix of LOCAL_HOST_SUFFIXES) {
      if (lower.endsWith(suffix)) {
        name = name.slice(0, -suffix.length).trim();
        break;
      }
    }
    return name || String(device.ip || "").trim();
  };

  const serviceGlyph = (service) => {
    const name = String(service.name || "");
    const product = String(service.product || "");
    for (const [pattern, glyph] of SERVICE_GLYPH_RULES) {
      if (pattern.test(name)) return glyph;
    }
    for (const [pattern, glyph] of SERVICE_GLYPH_RULES) {
      if (pattern.test(product)) return glyph;
    }
    return CATEGORY_GLYPHS[service.category] || "dot";
  };

  const glyphSVG = (glyph) => {
    const shapes = SERVICE_GLYPHS[glyph] || SERVICE_GLYPHS.dot;
    const body = shapes
      .map(([kind, ...values]) => {
        switch (kind) {
          case "ring":
            return `<circle cx="${values[0]}" cy="${values[1]}" r="${values[2]}"/>`;
          case "ellipse":
            return `<ellipse cx="${values[0]}" cy="${values[1]}" rx="${values[2]}" ry="${values[3]}"/>`;
          case "dot":
            return `<circle cx="${values[0]}" cy="${values[1]}" r="${values[2]}" fill="currentColor" stroke="none"/>`;
          case "rect":
            return `<rect x="${values[0]}" y="${values[1]}" width="${values[2]}" height="${values[3]}" rx="${values[4]}"/>`;
          case "fill":
            return `<path d="${values[0]}" fill="currentColor" stroke="none"/>`;
          default:
            return `<path d="${values[0]}"/>`;
        }
      })
      .join("");
    return `<svg viewBox="-4 -4 8 8" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">${body}</svg>`;
  };

  const rgbCache = new Map();
  const rgba = (hex, alpha) => {
    let rgb = rgbCache.get(hex);
    if (!rgb) {
      const value = hex.replace("#", "");
      rgb = [
        parseInt(value.slice(0, 2), 16),
        parseInt(value.slice(2, 4), 16),
        parseInt(value.slice(4, 6), 16),
      ];
      rgbCache.set(hex, rgb);
    }
    return `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, ${alpha})`;
  };

  const HOUSE_SVG = '<svg viewBox="-4 -4 8 8" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><path d="M -3.3 0.6 L 0 -2.8 L 3.3 0.6"/><path d="M -2.5 -0.1 V 3.3 H 2.5 V -0.1"/></svg>';
  const KNOT_SVG = '<svg viewBox="-4 -4 8 8" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><path d="M -2.7 2.2 L 0 -2.9 L 2.7 2.2 Z"/><circle cx="-2.7" cy="2.2" r=".85" fill="currentColor" stroke="none"/><circle cx="0" cy="-2.9" r=".85" fill="currentColor" stroke="none"/><circle cx="2.7" cy="2.2" r=".85" fill="currentColor" stroke="none"/></svg>';

  const hasLocal = (device) => (device.local === undefined ? !device.tailnet : Boolean(device.local));
  const hasTailnet = (device) => Boolean(device.tailnet);
  const deviceReach = (device) => {
    if (hasLocal(device) && hasTailnet(device)) return "both";
    if (hasTailnet(device)) return "tailnet";
    return "local";
  };
  const reachLabel = (reach) => {
    if (reach === "both") return "LAN + Tailnet";
    if (reach === "tailnet") return "Tailnet";
    return "LAN";
  };
  const reachShort = (reach) => {
    if (reach === "both") return "L+T";
    if (reach === "tailnet") return "TS";
    return "LAN";
  };

  const validTime = (value) => {
    const parsed = Date.parse(value || "");
    return Number.isFinite(parsed) && parsed > 946684800000;
  };
  const relativeTime = (value) => {
    if (!validTime(value)) return "never";
    const seconds = Math.max(0, Math.round((Date.now() - Date.parse(value)) / 1000));
    if (seconds < 10) return "just now";
    if (seconds < 60) return `${seconds}s ago`;
    const minutes = Math.round(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.round(minutes / 60);
    if (hours < 48) return `${hours}h ago`;
    return `${Math.round(hours / 24)}d ago`;
  };
  // Map labels stay short, so seconds collapse into "just now".
  const shortRelativeTime = (value) =>
    (validTime(value) && Date.now() - Date.parse(value) < 60000 ? "just now" : relativeTime(value));
  // Formatted during each draw, so the text stays current between scans.
  const lastSeenLabel = (device) =>
    (validTime(device.lastSeen) ? `last seen ${shortRelativeTime(device.lastSeen)}` : "history");
  const absoluteTime = (value) =>
    (validTime(value)
      ? new Date(value).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })
      : "unknown");
  // One source for a service row's visible state and its accessible timestamp.
  const serviceState = (service) => {
    if (!validTime(service.lastSeen)) {
      return { label: "history", title: "No recorded last seen", tone: "off" };
    }
    if (service.online) {
      return { label: "active", title: `Active, last seen ${absoluteTime(service.lastSeen)}`, tone: "on" };
    }
    return { label: `last seen ${shortRelativeTime(service.lastSeen)}`, title: `Last seen ${absoluteTime(service.lastSeen)}`, tone: "off" };
  };

  const canvas = document.getElementById("map");
  const ctx = canvas.getContext("2d");
  const panel = document.getElementById("panel");
  const tooltip = document.getElementById("tooltip");
  const toast = document.getElementById("toast");
  const subtitle = document.getElementById("subtitle");
  const stats = document.getElementById("stats");
  const historyBtn = document.getElementById("historyBtn");
  const tailnetBtn = document.getElementById("tailnetBtn");
  const scanBtn = document.getElementById("scanBtn");
  const settingsBtn = document.getElementById("settingsBtn");
  const settingsDialog = document.getElementById("settingsDialog");
  const settingsClose = document.getElementById("settingsClose");
  const settingsHistoryBtn = document.getElementById("settingsHistoryBtn");
  const settingsTailnetBtn = document.getElementById("settingsTailnetBtn");
  const settingsResetBtn = document.getElementById("settingsResetBtn");
  const conn = document.getElementById("conn");
  const legend = document.getElementById("legend");
  const empty = document.getElementById("empty");
  const deviceList = document.getElementById("deviceList");
  const deviceListItems = document.getElementById("deviceListItems");
  const deviceListClose = document.getElementById("deviceListClose");
  const deviceSummary = document.getElementById("deviceSummary");
  const devicesBtn = document.getElementById("devicesBtn");

  const size = {
    width: window.innerWidth,
    height: window.innerHeight,
    dpr: Math.min(window.devicePixelRatio || 1, 2),
    chromeTop: 94,
    insetRight: 0,
  };

  const view = {
    snapshot: null,
    nodes: new Map(),
    links: [],
    camera: { x: 0, y: 0, zoom: 1 },
    target: null,
    hoverId: null,
    selectedDeviceId: null,
    selectedServiceId: null,
    showOffline: true,
    showTailnet: true,
    dockDetails: false,
    deviceSheetOpen: false,
    panelReturnFocus: null,
    tailnetPresent: false,
    territories: null,
    territoryVersion: 0,
    territorySig: "",
    territoryLayer: null,
    dragging: false,
    dragMoved: false,
    dragStart: null,
    dragNodeId: null,
    nodeDragStart: null,
    pointers: new Map(),
    pinchDistance: 0,
    lastPinchCenter: null,
    pointerDownHit: null,
    pointer: { x: 0, y: 0 },
    initialized: false,
    userAdjusted: false,
    reducedMotion: window.matchMedia("(prefers-reduced-motion: reduce)").matches,
    lastErrorShown: "",
    toastTimer: 0,
    // Live traffic edges keyed by "deviceId|via". Entries fade out rather
    // than vanish when a flow stops.
    traffic: new Map(),
  };

  const deviceSummaryState = {
    announced: "",
    pending: "",
    timer: 0,
    announcedAt: 0,
  };

  try {
    if (window.localStorage.getItem("hearthlight.history") === "0") {
      view.showOffline = false;
    }
    if (window.localStorage.getItem("hearthlight.tailnet") === "0") {
      view.showTailnet = false;
    }
    if (window.localStorage.getItem("hearthlight.dock") === "1") {
      view.dockDetails = true;
    }
  } catch (_error) {
    // Private browsing can disable localStorage; history stays on.
  }

  function tailnetShown() {
    return view.showTailnet && view.tailnetPresent;
  }

  function deviceVisible(device) {
    if (hasTailnet(device) && !hasLocal(device) && !view.showTailnet) return false;
    return Boolean(device.online) || view.showOffline;
  }

  // A pinned or actively dragged device holds its manual map position while
  // the rest of the automatic layout settles around it.
  function deviceAnchored(node) {
    return Boolean(node.data.pinned) || node.id === view.dragNodeId;
  }

  // The automatic layout seeds every host from its id, then physics settles
  // the cluster. Reset reuses the same seed so a cleared map returns to the
  // placement it had before the first pin.
  function automaticDeviceOrigin(device) {
    const seed = hashString(device.id);
    const angle = ((seed % 3600) / 3600) * TAU;
    const radius = 120 + (seed % 240);
    const reach = deviceReach(device);
    const bias = !view.tailnetPresent ? 0 : reach === "local" ? -600 : reach === "tailnet" ? 620 : 0;
    return {
      x: Math.cos(angle) * radius * 0.8 + bias,
      y: Math.sin(angle) * radius * 0.5,
      seed: (seed % 1000) / 1000,
    };
  }

  function dockEnabled() {
    return view.dockDetails && size.width >= DOCK_MIN_WIDTH;
  }

  function dockedOpen() {
    return dockEnabled() && panel.classList.contains("open");
  }

  function syncPanelDock() {
    panel.classList.toggle("docked", dockEnabled());
  }

  function updateMapInset() {
    size.insetRight = dockedOpen() ? panel.getBoundingClientRect().width : 0;
  }

  function resize() {
    size.width = window.innerWidth;
    size.height = window.innerHeight;
    size.dpr = Math.min(window.devicePixelRatio || 1, 2);
    const topbar = document.querySelector(".topbar");
    size.chromeTop = topbar ? topbar.getBoundingClientRect().bottom + 10 : 94;
    syncPanelDock();
    updateMapInset();
    canvas.width = Math.round(size.width * size.dpr);
    canvas.height = Math.round(size.height * size.dpr);
    if (view.snapshot && !view.userAdjusted) {
      fitView(false);
    }
    ensureSelectedVisible();
  }

  function worldToScreen(x, y) {
    return {
      x: (x - view.camera.x) * view.camera.zoom + (size.width - size.insetRight) / 2,
      y: (y - view.camera.y) * view.camera.zoom + size.height / 2,
    };
  }

  function screenToWorld(x, y) {
    return {
      x: (x - (size.width - size.insetRight) / 2) / view.camera.zoom + view.camera.x,
      y: (y - size.height / 2) / view.camera.zoom + view.camera.y,
    };
  }

  function zoomAt(screenX, screenY, factor) {
    const before = screenToWorld(screenX, screenY);
    const next = clamp(view.camera.zoom * factor, 0.2, 3.2);
    if (next === view.camera.zoom) return;
    view.camera.zoom = next;
    const after = screenToWorld(screenX, screenY);
    view.camera.x += before.x - after.x;
    view.camera.y += before.y - after.y;
    view.target = null;
    view.userAdjusted = true;
  }

  function fitView(animate) {
    const devices = [...view.nodes.values()].filter(
      (node) => node.type === "device" && deviceVisible(node.data),
    );
    if (!devices.length) {
      const target = { x: 0, y: 0, zoom: 1 };
      if (animate) {
        view.target = target;
      } else {
        Object.assign(view.camera, target);
        view.target = null;
      }
      view.userAdjusted = false;
      return;
    }

    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    for (const node of devices) {
      const pad = node.baseR + ((node.data.services || []).length ? 58 : 32) + (view.tailnetPresent ? 34 : 0);
      minX = Math.min(minX, node.x - pad);
      minY = Math.min(minY, node.y - pad);
      maxX = Math.max(maxX, node.x + pad);
      maxY = Math.max(maxY, node.y + pad);
    }

    const width = Math.max(1, maxX - minX);
    const height = Math.max(1, maxY - minY);
    const viewportWidth = Math.max(1, size.width - size.insetRight);
    const narrow = size.width < 700;
    const horizontalPad = narrow ? 26 : 90;
    const verticalPad = narrow ? Math.max(260, size.chromeTop + 150) : 230;
    const zoom = clamp(
      Math.min((viewportWidth - horizontalPad) / width, (size.height - verticalPad) / height),
      narrow ? 0.2 : 0.28,
      1.55,
    );
    const target = {
      x: (minX + maxX) / 2,
      y: (minY + maxY) / 2,
      zoom,
    };
    if (animate) {
      view.target = target;
    } else {
      Object.assign(view.camera, target);
      view.target = null;
    }
    view.userAdjusted = false;
    ensureSelectedVisible();
  }

  function updateCamera(dt) {
    if (!view.target) return;
    const factor = 1 - Math.exp(-dt * 9);
    view.camera.x = lerp(view.camera.x, view.target.x, factor);
    view.camera.y = lerp(view.camera.y, view.target.y, factor);
    view.camera.zoom = lerp(view.camera.zoom, view.target.zoom, factor);
    if (
      Math.abs(view.camera.x - view.target.x) < 0.05 &&
      Math.abs(view.camera.y - view.target.y) < 0.05 &&
      Math.abs(view.camera.zoom - view.target.zoom) < 0.0005
    ) {
      view.camera.x = view.target.x;
      view.camera.y = view.target.y;
      view.camera.zoom = view.target.zoom;
      view.target = null;
    }
  }

  function serviceAlpha(node, zoom) {
    if (!node.data.online) {
      if (!view.showOffline) return 0;
      return smoothstep(0.9, 1.2, zoom) * 0.65;
    }
    const floor = view.tailnetPresent ? 0.42 : 0.16;
    return floor + (1 - floor) * smoothstep(0.5, 0.9, zoom);
  }

  function serviceLabelAlpha(node, zoom) {
    const base = smoothstep(0.95, 1.2, zoom);
    return node.data.online ? base : base * 0.75;
  }

  function deviceHistoricAlpha(node, zoom) {
    if (node.data.online) return 1;
    if (!view.showOffline) return 0;
    return 0.5 + 0.3 * smoothstep(0.8, 1.3, zoom);
  }

  // --- Aurora territories -------------------------------------------------
  //
  // Territories are isocontours over a smooth distance field, not convex hulls.
  // Every host owns a wide elliptical lobe (wide because the map lives on a
  // landscape screen). A minimum spanning tree joins nearby hosts with softly
  // bowed capsules, which gives the membrane its lasso-like necks instead of
  // ballooning across the empty middle of the cluster. Marching squares traces
  // the zero contour; two Chaikin passes round off the grid corners.
  //
  // The approach follows the isocontour idea from Bubble Sets and the classic
  // metaball + marching-squares technique:
  //   Collins et al., "Bubble Sets" (2009)
  //   http://sheelaghcarpendale.ca/innovis/uploads/Publications/Publications/Collins_2009_BubbleSets.pdf
  //   https://jamie-wong.com/2014/08/19/metaballs-and-marching-squares/

  const TERRITORY_BLEND = 44;
  const TERRITORY_BEND = 64;
  const TERRITORY_PAD_X = 104;
  const TERRITORY_PAD_Y = 80;
  const TERRITORY_NECK_X = 80;
  const TERRITORY_NECK_Y = 64;

  const MARCHING_PAIRS = [
    [], [["left", "top"]], [["top", "right"]], [["left", "right"]],
    [["right", "bottom"]], [["left", "top"], ["right", "bottom"]], [["top", "bottom"]], [["left", "bottom"]],
    [["bottom", "left"]], [["top", "bottom"]], [["top", "right"], ["bottom", "left"]], [["right", "bottom"]],
    [["right", "left"]], [["top", "right"]], [["left", "top"]], [],
  ];

  function minimumSpanningEdges(nodes) {
    if (nodes.length < 2) return [];
    const edges = [];
    const inTree = new Uint8Array(nodes.length);
    const best = new Float64Array(nodes.length).fill(Infinity);
    const parent = new Int32Array(nodes.length).fill(-1);
    let current = 0;
    inTree[current] = 1;
    for (let added = 0; added < nodes.length - 1; added += 1) {
      const from = nodes[current];
      for (let i = 0; i < nodes.length; i += 1) {
        if (inTree[i]) continue;
        const distance = Math.hypot(nodes[i].x - from.x, nodes[i].y - from.y);
        if (distance < best[i]) {
          best[i] = distance;
          parent[i] = current;
        }
      }
      let next = -1;
      let nextDistance = Infinity;
      for (let i = 0; i < nodes.length; i += 1) {
        if (!inTree[i] && best[i] < nextDistance) {
          next = i;
          nextDistance = best[i];
        }
      }
      if (next < 0) break;
      inTree[next] = 1;
      edges.push({ a: nodes[parent[next]], b: nodes[next] });
      current = next;
    }
    return edges;
  }

  // territoryShapes builds the field primitives: one lobe per host and a bowed
  // capsule per tree edge. The bow is stable per pair, so the membrane does not
  // twitch when a snapshot refreshes.
  function territoryShapes(nodes) {
    const shapes = [];
    for (const node of nodes) {
      shapes.push({
        x: node.x,
        y: node.y,
        rx: node.baseR + TERRITORY_PAD_X,
        ry: node.baseR + TERRITORY_PAD_Y,
        core: node.baseR + TERRITORY_PAD_Y,
      });
    }
    for (const edge of minimumSpanningEdges(nodes)) {
      const a = edge.a;
      const b = edge.b;
      const length = Math.hypot(b.x - a.x, b.y - a.y);
      if (length < 1) continue;
      const bend = clamp(length * 0.11, 10, TERRITORY_BEND) *
        (hashString(`${a.id}:${b.id}`) % 2 ? 1 : -1);
      const nx = -(b.y - a.y) / length;
      const ny = (b.x - a.x) / length;
      const control = {
        x: (a.x + b.x) / 2 + nx * bend,
        y: (a.y + b.y) / 2 + ny * bend,
      };
      const steps = Math.max(3, Math.round(length / 90));
      let previous = { x: a.x, y: a.y };
      for (let i = 1; i <= steps; i += 1) {
        const t = i / steps;
        const mt = 1 - t;
        const point = {
          x: mt * mt * a.x + 2 * mt * t * control.x + t * t * b.x,
          y: mt * mt * a.y + 2 * mt * t * control.y + t * t * b.y,
        };
        shapes.push({
          x1: previous.x,
          y1: previous.y,
          x2: point.x,
          y2: point.y,
          rx: TERRITORY_NECK_X,
          ry: TERRITORY_NECK_Y,
          core: TERRITORY_NECK_Y,
        });
        previous = point;
      }
    }
    return shapes;
  }

  function fieldDistance(shape, x, y) {
    if (shape.x1 === undefined) {
      const dx = (x - shape.x) / shape.rx;
      const dy = (y - shape.y) / shape.ry;
      return (Math.hypot(dx, dy) - 1) * shape.core;
    }
    const vx = shape.x2 - shape.x1;
    const vy = shape.y2 - shape.y1;
    const lengthSq = vx * vx + vy * vy || 1;
    const t = clamp(((x - shape.x1) * vx + (y - shape.y1) * vy) / lengthSq, 0, 1);
    const dx = (x - (shape.x1 + vx * t)) / shape.rx;
    const dy = (y - (shape.y1 + vy * t)) / shape.ry;
    return (Math.hypot(dx, dy) - 1) * shape.core;
  }

  function smoothMin(a, b) {
    const h = Math.max(TERRITORY_BLEND - Math.abs(a - b), 0) / TERRITORY_BLEND;
    return Math.min(a, b) - h * h * TERRITORY_BLEND * 0.25;
  }

  function sampleTerritoryField(shapes, bounds, cell) {
    const cols = Math.max(2, Math.ceil((bounds.maxX - bounds.minX) / cell) + 1);
    const rows = Math.max(2, Math.ceil((bounds.maxY - bounds.minY) / cell) + 1);
    const values = new Float64Array(cols * rows);
    const boxes = shapes.map((shape) => {
      if (shape.x1 === undefined) {
        return [
          shape.x - shape.rx, shape.y - shape.ry,
          shape.x + shape.rx, shape.y + shape.ry,
        ];
      }
      return [
        Math.min(shape.x1, shape.x2) - shape.rx,
        Math.min(shape.y1, shape.y2) - shape.ry,
        Math.max(shape.x1, shape.x2) + shape.rx,
        Math.max(shape.y1, shape.y2) + shape.ry,
      ];
    });
    for (let row = 0; row < rows; row += 1) {
      const y = bounds.minY + row * cell;
      for (let col = 0; col < cols; col += 1) {
        const x = bounds.minX + col * cell;
        let value = Infinity;
        for (let i = 0; i < shapes.length; i += 1) {
          const box = boxes[i];
          // Outside every lobe's padded bounds the field is positive and cannot
          // move the zero contour, so skip the expensive distance test.
          if (x < box[0] - TERRITORY_BLEND || x > box[2] + TERRITORY_BLEND ||
            y < box[1] - TERRITORY_BLEND || y > box[3] + TERRITORY_BLEND) {
            continue;
          }
          const distance = fieldDistance(shapes[i], x, y);
          if (value === Infinity) {
            value = distance;
          } else if (distance < value + TERRITORY_BLEND) {
            value = smoothMin(value, distance);
          }
        }
        values[row * cols + col] = value === Infinity ? TERRITORY_BLEND + 1 : value;
      }
    }
    return { values, cols, rows };
  }

  function stitchContours(segments, edgePoints) {
    const adjacency = new Map();
    segments.forEach((segment, index) => {
      for (const key of segment) {
        const list = adjacency.get(key);
        if (list) list.push(index);
        else adjacency.set(key, [index]);
      }
    });
    const used = new Uint8Array(segments.length);
    const loops = [];
    for (let start = 0; start < segments.length; start += 1) {
      if (used[start]) continue;
      const keys = [];
      let segmentIndex = start;
      let key = segments[start][0];
      const first = key;
      let guard = 0;
      while (segmentIndex !== -1 && guard <= segments.length) {
        used[segmentIndex] = 1;
        const segment = segments[segmentIndex];
        keys.push(key);
        key = segment[0] === key ? segment[1] : segment[0];
        if (key === first) break;
        const candidates = adjacency.get(key) || [];
        segmentIndex = -1;
        for (const candidate of candidates) {
          if (!used[candidate]) {
            segmentIndex = candidate;
            break;
          }
        }
        guard += 1;
      }
      if (key === first && keys.length >= 4) {
        loops.push(keys.map((pointKey) => edgePoints.get(pointKey)));
      }
    }
    return loops;
  }

  function traceContours(field, bounds, cell) {
    const { values, cols, rows } = field;
    const edgePoints = new Map();
    const segments = [];

    const edgeKey = (name, col, row) => {
      if (name === "top") return `H:${col}:${row}`;
      if (name === "bottom") return `H:${col}:${row + 1}`;
      if (name === "left") return `V:${col}:${row}`;
      return `V:${col + 1}:${row}`;
    };

    const edgePoint = (name, col, row) => {
      const key = edgeKey(name, col, row);
      if (!edgePoints.has(key)) {
        let v1;
        let v2;
        let gx;
        let gy;
        let horizontal;
        if (name === "top") {
          v1 = values[row * cols + col];
          v2 = values[row * cols + col + 1];
          gx = col;
          gy = row;
          horizontal = true;
        } else if (name === "bottom") {
          v1 = values[(row + 1) * cols + col];
          v2 = values[(row + 1) * cols + col + 1];
          gx = col;
          gy = row + 1;
          horizontal = true;
        } else if (name === "left") {
          v1 = values[row * cols + col];
          v2 = values[(row + 1) * cols + col];
          gx = col;
          gy = row;
          horizontal = false;
        } else {
          v1 = values[row * cols + col + 1];
          v2 = values[(row + 1) * cols + col + 1];
          gx = col + 1;
          gy = row;
          horizontal = false;
        }
        const denominator = v1 - v2;
        const t = Math.abs(denominator) < 1e-9 ? 0.5 : clamp(v1 / denominator, 0, 1);
        edgePoints.set(key, {
          x: bounds.minX + (gx + (horizontal ? t : 0)) * cell,
          y: bounds.minY + (gy + (horizontal ? 0 : t)) * cell,
        });
      }
      return key;
    };

    for (let row = 0; row < rows - 1; row += 1) {
      for (let col = 0; col < cols - 1; col += 1) {
        const a = values[row * cols + col];
        const b = values[row * cols + col + 1];
        const c = values[(row + 1) * cols + col + 1];
        const d = values[(row + 1) * cols + col];
        const mask = (a < 0 ? 1 : 0) | (b < 0 ? 2 : 0) | (c < 0 ? 4 : 0) | (d < 0 ? 8 : 0);
        for (const [from, to] of MARCHING_PAIRS[mask]) {
          segments.push([edgePoint(from, col, row), edgePoint(to, col, row)]);
        }
      }
    }
    return stitchContours(segments, edgePoints);
  }

  function loopArea(points) {
    let area = 0;
    for (let i = 0; i < points.length; i += 1) {
      const a = points[i];
      const b = points[(i + 1) % points.length];
      area += a.x * b.y - b.x * a.y;
    }
    return area / 2;
  }

  function pointInLoop(point, loop) {
    let inside = false;
    for (let i = 0, j = loop.length - 1; i < loop.length; j = i, i += 1) {
      const a = loop[i];
      const b = loop[j];
      if ((a.y > point.y) !== (b.y > point.y) &&
        point.x < ((b.x - a.x) * (point.y - a.y)) / (b.y - a.y) + a.x) {
        inside = !inside;
      }
    }
    return inside;
  }

  // outerLoops drops holes. A territory is a solid membership field, so an
  // empty pocket inside the ring of hosts should still read as territory.
  function outerLoops(loops) {
    const sorted = loops.slice().sort((a, b) => Math.abs(loopArea(b)) - Math.abs(loopArea(a)));
    const kept = [];
    for (const loop of sorted) {
      if (!kept.some((outer) => pointInLoop(loop[0], outer))) kept.push(loop);
    }
    return kept;
  }

  // chaikinLoop cuts every corner at 1/4 and 3/4, which turns the marching
  // squares polyline into gentle, cloud-like curves without overshoot.
  function chaikinLoop(points, passes) {
    let current = points;
    for (let pass = 0; pass < passes; pass += 1) {
      const next = [];
      for (let i = 0; i < current.length; i += 1) {
        const a = current[i];
        const b = current[(i + 1) % current.length];
        next.push({ x: a.x * 0.75 + b.x * 0.25, y: a.y * 0.75 + b.y * 0.25 });
        next.push({ x: a.x * 0.25 + b.x * 0.75, y: a.y * 0.25 + b.y * 0.75 });
      }
      current = next;
    }
    return current;
  }

  function territoryBounds(shapes) {
    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    for (const shape of shapes) {
      if (shape.x1 === undefined) {
        minX = Math.min(minX, shape.x - shape.rx);
        minY = Math.min(minY, shape.y - shape.ry);
        maxX = Math.max(maxX, shape.x + shape.rx);
        maxY = Math.max(maxY, shape.y + shape.ry);
      } else {
        minX = Math.min(minX, Math.min(shape.x1, shape.x2) - shape.rx);
        minY = Math.min(minY, Math.min(shape.y1, shape.y2) - shape.ry);
        maxX = Math.max(maxX, Math.max(shape.x1, shape.x2) + shape.rx);
        maxY = Math.max(maxY, Math.max(shape.y1, shape.y2) + shape.ry);
      }
    }
    minX -= TERRITORY_BEND;
    minY -= TERRITORY_BEND;
    maxX += TERRITORY_BEND;
    maxY += TERRITORY_BEND;
    return { minX, minY, maxX, maxY };
  }

  function buildTerritory(nodes) {
    if (!nodes.length) return null;
    const shapes = territoryShapes(nodes);
    const raw = territoryBounds(shapes);
    const span = Math.max(raw.maxX - raw.minX, raw.maxY - raw.minY);
    const cell = clamp(span / 160, 6, 12);
    const margin = cell * 3 + TERRITORY_BLEND;
    const bounds = {
      minX: raw.minX - margin,
      minY: raw.minY - margin,
      maxX: raw.maxX + margin,
      maxY: raw.maxY + margin,
    };
    const field = sampleTerritoryField(shapes, bounds, cell);
    const minArea = cell * cell * 4;
    const loops = outerLoops(
      traceContours(field, bounds, cell)
        .map((points) => chaikinLoop(points, 2))
        .filter((points) => Math.abs(loopArea(points)) > minArea),
    );
    if (!loops.length) return null;
    return { loops, bounds: pointsBounds(loops.flat()), count: nodes.length };
  }

  function updateTerritories() {
    const devices = [...view.nodes.values()].filter((node) => node.type === "device");
    const local = devices.filter((node) => hasLocal(node.data));
    const tailnet = devices.filter((node) => hasTailnet(node.data));
    view.territories = {
      lan: local.length ? buildTerritory(local) : null,
      tailnet: tailnet.length ? buildTerritory(tailnet) : null,
      lanCount: local.length,
      tailnetCount: tailnet.length,
    };
    view.territoryVersion += 1;
    view.territorySig = "";
  }

  function screenTerritory(territory) {
    const loops = territory.loops.map((points) =>
      points.map((point) => worldToScreen(point.x, point.y)));
    return { loops, bounds: pointsBounds(loops.flat()), count: territory.count };
  }

  function pointsBounds(points) {
    let minX = Infinity;
    let minY = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    for (const point of points) {
      minX = Math.min(minX, point.x);
      minY = Math.min(minY, point.y);
      maxX = Math.max(maxX, point.x);
      maxY = Math.max(maxY, point.y);
    }
    return { minX, minY, maxX, maxY };
  }

  function traceLoop(target, points) {
    if (!points || points.length < 3) return;
    target.moveTo(points[0].x, points[0].y);
    for (let i = 1; i < points.length; i += 1) {
      target.lineTo(points[i].x, points[i].y);
    }
    target.closePath();
  }

  function traceLoops(target, loops) {
    for (const loop of loops) traceLoop(target, loop);
  }

  function drawTerritoryFill(target, territory, tailnet) {
    const bounds = territory.bounds;
    const gradient = target.createLinearGradient(bounds.minX, bounds.minY, bounds.maxX, bounds.maxY);
    if (tailnet) {
      gradient.addColorStop(0, "rgba(94, 231, 255, 0.22)");
      gradient.addColorStop(0.5, "rgba(177, 140, 255, 0.14)");
      gradient.addColorStop(1, "rgba(94, 224, 165, 0.05)");
    } else {
      gradient.addColorStop(0, "rgba(255, 226, 166, 0.24)");
      gradient.addColorStop(0.55, "rgba(255, 209, 102, 0.12)");
      gradient.addColorStop(1, "rgba(94, 231, 255, 0.04)");
    }
    target.save();
    if (typeof target.filter === "string") target.filter = "blur(26px)";
    target.fillStyle = gradient;
    target.beginPath();
    traceLoops(target, territory.loops);
    target.fill("evenodd");
    target.restore();

    target.save();
    target.fillStyle = tailnet ? "rgba(94, 231, 255, 0.028)" : "rgba(255, 209, 102, 0.022)";
    target.beginPath();
    traceLoops(target, territory.loops);
    target.fill("evenodd");
    target.restore();
  }

  function drawAuroraRibbon(target, lan, tailnet) {
    const from = lan.bounds;
    const to = tailnet.bounds;
    const start = { x: (from.minX + from.maxX) / 2, y: (from.minY + from.maxY) / 2 - 40 };
    const end = { x: (to.minX + to.maxX) / 2, y: (to.minY + to.maxY) / 2 - 30 };
    const gradient = target.createLinearGradient(start.x, 0, end.x, 0);
    gradient.addColorStop(0, "rgba(94, 231, 255, 0.2)");
    gradient.addColorStop(0.55, "rgba(177, 140, 255, 0.13)");
    gradient.addColorStop(1, "rgba(94, 231, 255, 0)");
    target.save();
    if (typeof target.filter === "string") target.filter = "blur(16px)";
    target.strokeStyle = gradient;
    target.lineWidth = 54;
    target.lineCap = "round";
    target.beginPath();
    target.moveTo(start.x, start.y);
    target.bezierCurveTo(
      lerp(start.x, end.x, 0.35), start.y - 70,
      lerp(start.x, end.x, 0.7), end.y + 55,
      end.x, end.y,
    );
    target.stroke();
    target.restore();
  }

  function drawOverlapLens(target, lan, tailnet) {
    const dual = [...view.nodes.values()].filter(
      (node) => node.type === "device" && hasLocal(node.data) && hasTailnet(node.data),
    );
    if (!dual.length) return;
    let cx = 0;
    let cy = 0;
    for (const node of dual) {
      const point = worldToScreen(node.x, node.y);
      cx += point.x;
      cy += point.y;
    }
    cx /= dual.length;
    cy /= dual.length;
    let rx = 150;
    let ry = 165;
    for (const node of dual) {
      const point = worldToScreen(node.x, node.y);
      rx = Math.max(rx, Math.abs(point.x - cx) + 100);
      ry = Math.max(ry, Math.abs(point.y - cy) + 105);
    }
    target.save();
    target.beginPath();
    traceLoops(target, lan.loops);
    target.clip("evenodd");
    target.beginPath();
    traceLoops(target, tailnet.loops);
    target.clip("evenodd");
    target.globalCompositeOperation = "screen";
    if (typeof target.filter === "string") target.filter = "blur(14px)";
    const glow = target.createRadialGradient(cx, cy, 0, cx, cy, Math.max(rx, ry));
    glow.addColorStop(0, "rgba(232, 255, 255, 0.26)");
    glow.addColorStop(0.46, "rgba(94, 231, 255, 0.1)");
    glow.addColorStop(1, "rgba(94, 231, 255, 0)");
    target.fillStyle = glow;
    target.fillRect(cx - rx * 1.7, cy - ry * 1.7, rx * 3.4, ry * 3.4);
    target.restore();
  }

  function renderTerritoryLayer() {
    const signature = [
      size.width, size.height, size.insetRight, view.camera.x.toFixed(2), view.camera.y.toFixed(2),
      view.camera.zoom.toFixed(4), view.territoryVersion, view.tailnetPresent, view.showTailnet,
    ].join("|");
    if (view.territoryLayer && view.territorySig === signature) return view.territoryLayer;
    const pixelWidth = Math.max(1, Math.round(size.width * size.dpr));
    const pixelHeight = Math.max(1, Math.round(size.height * size.dpr));
    let layer = view.territoryLayer;
    if (!layer || layer.width !== pixelWidth || layer.height !== pixelHeight) {
      layer = document.createElement("canvas");
      layer.width = pixelWidth;
      layer.height = pixelHeight;
    }
    const target = layer.getContext("2d");
    target.setTransform(1, 0, 0, 1, 0, 0);
    target.clearRect(0, 0, pixelWidth, pixelHeight);
    target.setTransform(size.dpr, 0, 0, size.dpr, 0, 0);
    const territories = view.territories;
    if (territories) {
      const lan = territories.lan ? screenTerritory(territories.lan) : null;
      const tailnet = territories.tailnet ? screenTerritory(territories.tailnet) : null;
      target.save();
      target.globalCompositeOperation = "screen";
      if (lan) drawTerritoryFill(target, lan, false);
      if (tailnet) drawTerritoryFill(target, tailnet, true);
      if (lan && tailnet) {
        drawAuroraRibbon(target, lan, tailnet);
        drawOverlapLens(target, lan, tailnet);
      }
      target.restore();
    }
    view.territoryLayer = layer;
    view.territorySig = signature;
    return layer;
  }

  function drawBoundary(territory, color, glow) {
    ctx.save();
    ctx.setLineDash([1.5, 7]);
    ctx.lineWidth = 1.2;
    ctx.strokeStyle = color;
    ctx.shadowColor = glow;
    ctx.shadowBlur = 9;
    ctx.beginPath();
    traceLoops(ctx, territory.loops);
    ctx.stroke();
    ctx.restore();
  }

  function drawTerritoryBoundaries() {
    if (!view.territories) return;
    if (view.territories.lan) {
      drawBoundary(screenTerritory(view.territories.lan), "rgba(255, 226, 166, 0.66)", "rgba(255, 209, 102, 0.42)");
    }
    if (view.territories.tailnet) {
      drawBoundary(screenTerritory(view.territories.tailnet), "rgba(126, 240, 255, 0.82)", "rgba(94, 231, 255, 0.62)");
    }
  }

  function topPoint(territory) {
    let best = null;
    for (const loop of territory.loops) {
      for (const point of loop) {
        const screen = worldToScreen(point.x, point.y);
        if (!best || screen.y < best.y) best = screen;
      }
    }
    return best || { x: size.width / 2, y: 120 };
  }

  function rectsOverlap(a, b, margin = 8) {
    return a.left < b.left + b.width + margin &&
      a.left + a.width + margin > b.left &&
      a.top < b.top + b.height + margin &&
      a.top + a.height + margin > b.top;
  }

  // deviceLabelRect reserves the space a host label will occupy so service
  // labels can dodge it. Device labels themselves always win.
  function deviceLabelRect(node, zoom) {
    if (!deviceVisible(node.data)) return null;
    const forced = view.hoverId === node.id || view.selectedDeviceId === node.id;
    const labelAlpha = forced ? 1 : smoothstep(0.26, 0.52, zoom);
    if (labelAlpha <= 0.01) return null;
    const device = node.data;
    const point = worldToScreen(node.x, node.y);
    const radius = node.baseR * zoom * (1 + node.hover * 0.06);
    const name = truncate(displayName(device), 24);
    const fontSize = Math.max(9.5, 13 * zoom);
    const textY = point.y + radius + 8 * zoom + 3;
    const reach = deviceReach(device);
    const subline = view.tailnetPresent && hasTailnet(device) && (reach === "tailnet" || device.self)
      ? (device.self ? `this host · ${device.tailnetIp || device.ip}` : (device.tailnetIp || device.ip))
      : "";
    ctx.save();
    ctx.font = `600 ${fontSize}px ${FONT}`;
    let width = ctx.measureText(name).width;
    let height = fontSize + 2;
    if (!device.online) {
      const subSize = Math.max(8.5, 10.5 * zoom);
      ctx.font = `500 ${subSize}px ${MONO}`;
      width = Math.max(width, ctx.measureText(lastSeenLabel(device)).width);
      height += subSize + 3;
    }
    if (subline) {
      const subSize = Math.max(8.5, 10.5 * zoom);
      ctx.font = `500 ${subSize}px ${MONO}`;
      width = Math.max(width, ctx.measureText(subline).width);
      height += subSize + 3;
    }
    ctx.restore();
    return { left: point.x - width / 2, top: textY, width, height };
  }

  function drawRegionPill(x, y, title, detail, tailnet, placed) {
    ctx.save();
    ctx.font = `600 9px ${MONO}`;
    const titleWidth = ctx.measureText(title).width;
    ctx.font = `500 9px ${MONO}`;
    const detailWidth = ctx.measureText(detail).width;
    const height = 24;
    const width = 30 + titleWidth + 9 + detailWidth;
    let left = clamp(x - width / 2, 18, size.width - width - 18);
    let top = Math.max(size.chromeTop + 2, y - height - 12);
    for (let attempt = 0; attempt < 3; attempt += 1) {
      const rect = { left, top, width, height };
      if (!placed.some((other) => rectsOverlap(other, rect))) break;
      top += 30;
    }
    top = Math.min(top, size.height - 110);

    ctx.shadowColor = tailnet ? "rgba(94, 231, 255, 0.14)" : "rgba(0, 0, 0, 0.45)";
    ctx.shadowBlur = 14;
    roundRectPath(left, top, width, height, height / 2);
    ctx.fillStyle = tailnet ? "rgba(7, 18, 28, 0.78)" : "rgba(7, 14, 24, 0.78)";
    ctx.fill();
    ctx.shadowBlur = 0;
    ctx.strokeStyle = tailnet ? "rgba(94, 231, 255, 0.32)" : "rgba(151, 196, 225, 0.2)";
    ctx.lineWidth = 1;
    ctx.stroke();

    ctx.beginPath();
    ctx.arc(left + 14, top + height / 2, 2.6, 0, TAU);
    ctx.fillStyle = tailnet ? "#5ee7ff" : "#dbe7f3";
    ctx.fill();

    ctx.textBaseline = "middle";
    ctx.textAlign = "left";
    ctx.font = `600 9px ${MONO}`;
    ctx.fillStyle = tailnet ? "#d7fbff" : "#e6eff8";
    ctx.fillText(title, left + 23, top + height / 2 + 0.5);
    ctx.font = `500 9px ${MONO}`;
    ctx.fillStyle = tailnet ? "#96dbe8" : "#8fa2b8";
    ctx.fillText(detail, left + 23 + titleWidth + 9, top + height / 2 + 0.5);
    ctx.restore();
    return { left, top, width, height };
  }

  function drawRegionLabels() {
    if (!view.territories || !view.snapshot) return;
    const { lan, tailnet, lanCount, tailnetCount } = view.territories;
    const network = view.snapshot.networks && view.snapshot.networks.length
      ? view.snapshot.networks[0]
      : "local network";
    const ssid = view.snapshot.ssid ? truncate(view.snapshot.ssid, 24) : "";
    const info = view.snapshot.tailnet;
    const suffix = info && (info.magicDnsSuffix || info.name) ? (info.magicDnsSuffix || info.name) : "tailnet";
    const peers = info && info.peersTotal ? info.peersTotal : tailnetCount;
    const placed = [];
    if (lan) {
      const top = topPoint(lan);
      const detail = [network, ssid, `${lanCount} HOSTS`].filter(Boolean).join(" · ");
      placed.push(drawRegionPill(top.x, top.y, "HOME LAN", detail, false, placed));
    }
    if (tailnet) {
      const top = topPoint(tailnet);
      placed.push(drawRegionPill(top.x, top.y, "TAILNET", `${suffix} · ${peers} PEERS`, true, placed));
    }
  }

  function drawReachBadge(point, radius, reach, alpha) {
    const scale = clamp(0.82 + view.camera.zoom * 0.2, 0.82, 1.04);
    const height = 17 * scale;
    const width = (reach === "both" ? 30 : 17) * scale;
    const cx = point.x - radius * 0.72;
    const cy = point.y - radius * 0.72;
    const left = cx - width / 2;
    const top = cy - height / 2;
    ctx.save();
    ctx.globalAlpha = alpha;
    roundRectPath(left, top, width, height, height / 2);
    ctx.fillStyle = "rgba(6, 15, 25, 0.94)";
    ctx.fill();
    ctx.strokeStyle = reach === "local" ? "rgba(219, 231, 243, 0.28)" : "rgba(94, 231, 255, 0.36)";
    ctx.lineWidth = 1;
    ctx.stroke();

    const glyph = height * 0.58;
    ctx.lineWidth = Math.max(0.8, height * 0.065);
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    if (reach === "tailnet" || reach === "both") {
      ctx.strokeStyle = "#5ee7ff";
      ctx.fillStyle = "#5ee7ff";
      traceKnot(reach === "both" ? left + width * 0.31 : cx, cy, glyph);
    }
    if (reach === "local" || reach === "both") {
      ctx.strokeStyle = "#dbe7f3";
      ctx.fillStyle = "#dbe7f3";
      traceHouse(reach === "both" ? left + width * 0.7 : cx, cy, glyph);
    }
    ctx.restore();
  }

  function traceHouse(x, y, size) {
    const u = size / 2;
    ctx.beginPath();
    ctx.moveTo(x - u * 0.85, y + u * 0.12);
    ctx.lineTo(x, y - u * 0.85);
    ctx.lineTo(x + u * 0.85, y + u * 0.12);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(x - u * 0.62, y - u * 0.04);
    ctx.lineTo(x - u * 0.62, y + u * 0.78);
    ctx.lineTo(x + u * 0.62, y + u * 0.78);
    ctx.lineTo(x + u * 0.62, y - u * 0.04);
    ctx.stroke();
  }

  function traceKnot(x, y, size) {
    const u = size / 2;
    ctx.beginPath();
    ctx.moveTo(x - u * 0.9, y + u * 0.72);
    ctx.lineTo(x, y - u * 0.92);
    ctx.lineTo(x + u * 0.9, y + u * 0.72);
    ctx.closePath();
    ctx.stroke();
    for (const [dx, dy] of [[-0.9, 0.72], [0, -0.92], [0.9, 0.72]]) {
      ctx.beginPath();
      ctx.arc(x + dx * u, y + dy * u, Math.max(0.8, u * 0.2), 0, TAU);
      ctx.fill();
    }
  }

  function physicsStep(dt) {
    const nodes = [...view.nodes.values()];
    const devices = nodes.filter((node) => node.type === "device");
    const services = nodes.filter((node) => node.type === "service");
    const displacement = new Map();
    for (const node of nodes) {
      displacement.set(node.id, { x: 0, y: 0 });
    }

    for (let i = 0; i < devices.length; i += 1) {
      for (let j = i + 1; j < devices.length; j += 1) {
        const a = devices[i];
        const b = devices[j];
        let dx = b.x - a.x;
        let dy = b.y - a.y;
        let distance = Math.hypot(dx, dy);
        if (distance < 0.001) {
          dx = 1;
          dy = 0;
          distance = 1;
        }
        const desired = (178 + (a.baseR + b.baseR) * 0.8) *
          (view.tailnetPresent && ((deviceReach(a.data) === "local" && deviceReach(b.data) === "tailnet") ||
            (deviceReach(a.data) === "tailnet" && deviceReach(b.data) === "local")) ? 1.22 : 1);
        if (distance < desired) {
          const push = ((desired - distance) / distance) * 0.32;
          const px = dx * push;
          const py = dy * push;
          if (!deviceAnchored(a)) {
            displacement.get(a.id).x -= px;
            displacement.get(a.id).y -= py;
          }
          if (!deviceAnchored(b)) {
            displacement.get(b.id).x += px;
            displacement.get(b.id).y += py;
          }
        }
      }
    }

    for (const node of services) {
      const parent = view.nodes.get(node.parentId);
      if (!parent) continue;
      const ring = parent.baseR + 36 + (node.ringIndex % 2) * 15;
      const targetX = parent.x + Math.cos(node.prefAngle) * ring;
      const targetY = parent.y + Math.sin(node.prefAngle) * ring;
      const move = displacement.get(node.id);
      move.x += (targetX - node.x) * 0.035;
      move.y += (targetY - node.y) * 0.035;
    }

    for (let i = 0; i < services.length; i += 1) {
      for (let j = i + 1; j < services.length; j += 1) {
        const a = services[i];
        const b = services[j];
        const dx = b.x - a.x;
        const dy = b.y - a.y;
        const distance = Math.hypot(dx, dy) || 0.001;
        const minimum = 22;
        if (distance < minimum) {
          const push = ((minimum - distance) / distance) * 0.45;
          displacement.get(a.id).x -= dx * push;
          displacement.get(a.id).y -= dy * push;
          displacement.get(b.id).x += dx * push;
          displacement.get(b.id).y += dy * push;
        }
      }
    }

    for (const service of services) {
      for (const device of devices) {
        if (device.id === service.parentId) continue;
        const dx = service.x - device.x;
        const dy = service.y - device.y;
        const distance = Math.hypot(dx, dy) || 0.001;
        const minimum = device.baseR + 24;
        if (distance < minimum) {
          const push = ((minimum - distance) / distance) * 0.5;
          displacement.get(service.id).x += dx * push;
          displacement.get(service.id).y += dy * push;
        }
      }
    }

    for (const device of devices) {
      if (deviceAnchored(device)) continue;
      const move = displacement.get(device.id);
      if (view.tailnetPresent) {
        // Bias the two memberships into left and right territories, with
        // dual-membership hosts held near the center so the lens sits between.
        const reach = deviceReach(device.data);
        const lane = reach === "local" ? -600 : reach === "tailnet" ? 620 : 0;
        const spread = reach === "both" ? 170 : 270;
        const targetX = lane + (device.seed - 0.5) * spread * 2;
        move.x += (targetX - device.x) * 0.012;
      }
      move.x -= device.x * 0.0012;
      // A gentle pull to the horizontal midline keeps each membership a
      // landscape-shaped archipelago instead of a tall column.
      move.y -= device.y * 0.0024;
    }

    const step = clamp(dt * 60, 0, 2);
    for (const node of nodes) {
      if (node.type === "device" && deviceAnchored(node)) continue;
      const move = displacement.get(node.id);
      node.x += move.x * step;
      node.y += move.y * step;
    }
  }

  function applySnapshot(snapshot) {
    if (!snapshot || !Array.isArray(snapshot.devices)) return;
    const wasTailnet = view.tailnetPresent;
    view.snapshot = snapshot;
    view.tailnetPresent = Boolean(snapshot.tailnet) || snapshot.devices.some((device) => hasTailnet(device));
    const presentDevices = new Set();
    const presentServices = new Set();

    for (const device of snapshot.devices) {
      presentDevices.add(device.id);
      let node = view.nodes.get(device.id);
      const savedX = Number.isFinite(device.x) ? device.x : 0;
      const savedY = Number.isFinite(device.y) ? device.y : 0;
      if (!node) {
        const origin = automaticDeviceOrigin(device);
        node = {
          id: device.id,
          type: "device",
          data: device,
          x: device.pinned ? savedX : origin.x,
          y: device.pinned ? savedY : origin.y,
          seed: origin.seed,
          baseR: 30,
          r: 30,
          hover: 0,
        };
        view.nodes.set(device.id, node);
      }
      node.data = device;
      if (device.pinned && node.id !== view.dragNodeId) {
        moveDeviceTo(node, savedX, savedY);
      }
      const serviceCount = (device.services || []).length;
      node.baseR = 30 + Math.min(16, Math.sqrt(serviceCount) * 4.4);
      node.r = node.baseR;

      let index = 0;
      for (const service of device.services || []) {
        presentServices.add(service.id);
        let serviceNode = view.nodes.get(service.id);
        if (!serviceNode) {
          const angle = node.seed * TAU + index * 2.399963;
          const ring = node.baseR + 38 + (index % 2) * 15;
          serviceNode = {
            id: service.id,
            type: "service",
            data: service,
            parentId: device.id,
            x: node.x + Math.cos(angle) * ring,
            y: node.y + Math.sin(angle) * ring,
            prefAngle: angle,
            ringIndex: index,
            seed: (hashString(service.id) % 1000) / 1000,
            baseR: SERVICE_R,
            r: SERVICE_R,
            hover: 0,
          };
          view.nodes.set(service.id, serviceNode);
        } else {
          serviceNode.data = service;
          serviceNode.parentId = device.id;
        }
        serviceNode.glyph = serviceGlyph(service);
        index += 1;
      }
    }

    for (const [id, node] of view.nodes) {
      if (node.type === "device" && !presentDevices.has(id)) {
        if (view.dragNodeId === id) clearNodeDrag();
        view.nodes.delete(id);
      } else if (node.type === "service" && !presentServices.has(id)) {
        view.nodes.delete(id);
      }
    }

    view.links = [];
    for (const node of view.nodes.values()) {
      if (node.type === "service") {
        view.links.push({ serviceId: node.id, deviceId: node.parentId });
      }
    }

    if (view.selectedDeviceId && !view.nodes.has(view.selectedDeviceId)) {
      view.selectedDeviceId = null;
      closePanel();
    }
    if (view.selectedServiceId && !view.nodes.has(view.selectedServiceId)) {
      view.selectedServiceId = null;
    }

    if (!view.initialized) {
      view.initialized = true;
      for (let i = 0; i < 300; i += 1) physicsStep(1 / 60);
      fitView(false);
    } else {
      for (let i = 0; i < 40; i += 1) physicsStep(1 / 60);
    }

    renderDeviceList(snapshot);
    announceDeviceSummary(snapshot);
    updateTerritories();
    hideTooltip();
    syncChrome();
    if (wasTailnet !== view.tailnetPresent) renderLegend();
    if (panel.classList.contains("open")) {
      renderPanel();
    }
  }

  function computeStats(devices) {
    let devicesOnline = 0;
    let servicesOnline = 0;
    let servicesTotal = 0;
    for (const device of devices) {
      if (device.online) devicesOnline += 1;
      for (const service of device.services || []) {
        servicesTotal += 1;
        if (service.online) servicesOnline += 1;
      }
    }
    return {
      devicesOnline,
      devicesTotal: devices.length,
      servicesOnline,
      servicesTotal,
    };
  }

  function renderDeviceList(snapshot) {
    if (!deviceListItems) return;
    const focused = deviceList && deviceList.contains(document.activeElement)
      ? document.activeElement.closest(".device-entry")
      : null;
    const focusedId = focused ? focused.dataset.deviceId : "";
    const focusedClose = Boolean(deviceList && document.activeElement === deviceListClose);
    const devices = [...snapshot.devices].sort((a, b) => {
      if (Boolean(a.online) !== Boolean(b.online)) return a.online ? -1 : 1;
      return displayName(a).localeCompare(displayName(b), undefined, { sensitivity: "base" });
    });
    if (!devices.length) {
      deviceListItems.innerHTML = '<li class="device-list-empty">No devices found yet.</li>';
      if (focusedClose) deviceListClose.focus();
      return;
    }
    deviceListItems.innerHTML = devices
      .map((device) => {
        const kind = KIND[device.kind] || KIND.device;
        const serviceCount = (device.services || []).length;
        const services = `${serviceCount} service${serviceCount === 1 ? "" : "s"}`;
        const ip = String(device.ip || "").trim() || "no address";
        return `
          <li>
            <button type="button" class="device-entry" data-device-id="${escapeHTML(device.id)}">
              <span class="device-entry-name">${escapeHTML(displayName(device))}</span>
              <span class="device-entry-meta">${device.online ? "online" : "history"} · ${escapeHTML(ip)} · ${escapeHTML(kind.label)} · ${services}</span>
            </button>
          </li>
        `;
      })
      .join("");
    if (focusedClose) {
      deviceListClose.focus();
    } else if (focusedId) {
      for (const entry of deviceListItems.querySelectorAll(".device-entry")) {
        if (entry.dataset.deviceId === focusedId) {
          entry.focus();
          break;
        }
      }
    }
  }

  function deviceEntries() {
    return deviceListItems ? [...deviceListItems.querySelectorAll(".device-entry")] : [];
  }

  function syncDeviceSheet() {
    if (deviceList) deviceList.classList.toggle("open", view.deviceSheetOpen);
    if (devicesBtn) devicesBtn.setAttribute("aria-expanded", String(view.deviceSheetOpen));
  }

  function openDeviceSheet() {
    if (!deviceList) return;
    view.deviceSheetOpen = true;
    syncDeviceSheet();
    const entries = deviceEntries();
    const selected = view.selectedDeviceId
      ? entries.find((entry) => entry.dataset.deviceId === view.selectedDeviceId)
      : null;
    const target = selected || entries[0] || deviceListClose || deviceList;
    if (target) target.focus();
  }

  function closeDeviceSheet(restoreFocus = true) {
    if (!deviceList) return;
    view.deviceSheetOpen = false;
    syncDeviceSheet();
    if (restoreFocus && devicesBtn) devicesBtn.focus();
  }

  function toggleDeviceSheet() {
    if (view.deviceSheetOpen) closeDeviceSheet();
    else openDeviceSheet();
  }

  function moveDeviceSheetFocus(event) {
    const entries = deviceEntries();
    if (!entries.length) return;
    const active = document.activeElement.closest(".device-entry");
    const index = entries.indexOf(active);
    if (index < 0) return;
    let next = index;
    switch (event.key) {
      case "ArrowDown":
        next = (index + 1) % entries.length;
        break;
      case "ArrowUp":
        next = (index - 1 + entries.length) % entries.length;
        break;
      case "Home":
        next = 0;
        break;
      case "End":
        next = entries.length - 1;
        break;
      default:
        return;
    }
    event.preventDefault();
    entries[next].focus();
  }

  // The live region is throttled and only written when its sentence changes,
  // so repeated snapshots do not re-announce the same counts.
  function announceDeviceSummary(snapshot) {
    if (!deviceSummary) return;
    const stats = snapshot.stats || computeStats(snapshot.devices);
    const plural = (count, word) => `${count} ${word}${count === 1 ? "" : "s"}`;
    const offline = Math.max(0, stats.devicesTotal - stats.devicesOnline);
    const scan = snapshot.lastError
      ? "scan failed"
      : snapshot.scanning
        ? "scan running"
        : validTime(snapshot.lastScan)
          ? "scan complete"
          : "waiting for first scan";
    const text = `${plural(stats.devicesTotal, "device")}, ${stats.devicesOnline} online, ${offline} in history. ${plural(stats.servicesTotal, "service")}, ${stats.servicesOnline} up. ${scan}.`;
    if (text === deviceSummaryState.announced) {
      deviceSummaryState.pending = "";
      return;
    }
    deviceSummaryState.pending = text;
    if (deviceSummaryState.timer) return;
    const wait = Math.max(0, DEVICE_SUMMARY_THROTTLE_MS - (Date.now() - deviceSummaryState.announcedAt));
    deviceSummaryState.timer = window.setTimeout(flushDeviceSummary, wait);
  }

  function flushDeviceSummary() {
    deviceSummaryState.timer = 0;
    const text = deviceSummaryState.pending;
    if (!text || text === deviceSummaryState.announced) return;
    deviceSummaryState.announced = text;
    deviceSummaryState.pending = "";
    deviceSummaryState.announcedAt = Date.now();
    deviceSummary.textContent = text;
  }

  function syncToggleState(button, on) {
    if (!button) return;
    button.setAttribute("aria-pressed", String(on));
    const state = button.querySelector(".settings-toggle-state");
    if (state) state.textContent = on ? "shown" : "hidden";
  }

  function syncSettingsControls() {
    syncToggleState(settingsHistoryBtn, view.showOffline);
    syncToggleState(settingsTailnetBtn, view.showTailnet);
  }

  function syncChrome() {
    const snapshot = view.snapshot;
    if (!snapshot) return;
    const summary = snapshot.stats || computeStats(snapshot.devices);
    const tailnetStat = snapshot.tailnet
      ? `<div class="stat tailnet"><b>${snapshot.tailnet.peersOnline}<em>/${snapshot.tailnet.peersTotal}</em></b><span>tailnet up</span></div>`
      : "";
    stats.innerHTML = `
      <div class="stat"><b>${summary.devicesOnline}<em>/${summary.devicesTotal}</em></b><span>devices up</span></div>
      <div class="stat"><b>${summary.servicesOnline}<em>/${summary.servicesTotal}</em></b><span>services up</span></div>
      ${tailnetStat}
      <div class="stat"><b>${snapshot.scanIntervalSeconds || 60}s</b><span>scan cadence</span></div>
    `;

    if (tailnetBtn) {
      tailnetBtn.hidden = !view.tailnetPresent;
      tailnetBtn.setAttribute("aria-pressed", String(view.showTailnet));
      tailnetBtn.title = view.showTailnet ? "Hide tailnet hosts and overlays" : "Show tailnet hosts and overlays";
    }

    scanBtn.disabled = Boolean(snapshot.scanning);
    scanBtn.dataset.scanning = String(Boolean(snapshot.scanning));
    scanBtn.querySelector("span:last-child").textContent = snapshot.scanning ? "Scanning" : "Scan";
    historyBtn.setAttribute("aria-pressed", String(view.showOffline));
    syncSettingsControls();

    if (snapshot.lastError && snapshot.lastError !== view.lastErrorShown) {
      view.lastErrorShown = snapshot.lastError;
      showToast(`Scan failed: ${snapshot.lastError}`);
    } else if (!snapshot.lastError) {
      view.lastErrorShown = "";
    }

    const noDevices = snapshot.devices.length === 0;
    empty.classList.toggle("show", noDevices);
    if (noDevices) {
      const heading = empty.querySelector("h2");
      const copy = empty.querySelector("p");
      if (snapshot.scanning) {
        heading.textContent = "Sweeping the network";
        copy.textContent = "Hearthlight is resolving neighbours and probing common services.";
      } else {
        heading.textContent = "No devices yet";
        copy.textContent = "Hearthlight is waiting for its first scan. Check the CIDR and make sure the service can reach the LAN.";
      }
    }

    updateSubtitle();
  }

  function updateSubtitle() {
    const snapshot = view.snapshot;
    if (!snapshot) return;
    const networks = snapshot.networks && snapshot.networks.length
      ? snapshot.networks.join(" · ")
      : "discovering local networks";
    const ssid = snapshot.ssid ? `Wi-Fi ${snapshot.ssid}` : "";
    const tailnet = snapshot.tailnet && (snapshot.tailnet.magicDnsSuffix || snapshot.tailnet.name)
      ? (snapshot.tailnet.magicDnsSuffix || snapshot.tailnet.name)
      : "";
    const scanned = validTime(snapshot.lastScan) ? `last scan ${relativeTime(snapshot.lastScan)}` : "waiting for first scan";
    const prefix = snapshot.scanning ? "scanning · " : "";
    const error = snapshot.lastError ? " · scan error" : "";
    subtitle.textContent = `${prefix}${[networks, ssid, tailnet, scanned].filter(Boolean).join(" · ")}${error}`;
    subtitle.title = snapshot.lastError || [networks, ssid, tailnet].filter(Boolean).join(" · ");
  }

  function renderLegend() {
    const kinds = Object.entries(KIND)
      .map(([, info]) => `<span class="legend-item"><i class="legend-dot" style="--c:${info.color}"></i>${info.label}</span>`)
      .join("");
    const categories = Object.entries(CATEGORY)
      .map(([name, color]) => `<span class="legend-item"><i class="legend-dot" style="--c:${color}"></i>${name}</span>`)
      .join("");
    const tailnet = view.tailnetPresent;
    const tailnetOnly = tailnet
      ? '<span class="legend-item"><i class="legend-halo"></i>tailnet-only</span>'
      : "";
    const reach = tailnet
      ? `<div class="legend-row">
        <span class="legend-label">reach</span>
        <span class="legend-item"><i class="legend-badge">${HOUSE_SVG}</i>local</span>
        <span class="legend-item"><i class="legend-badge tail">${KNOT_SVG}</i>tailnet</span>
        <span class="legend-item"><i class="legend-badge both">${HOUSE_SVG}${KNOT_SVG}</i>both</span>
        <span class="legend-item"><i class="legend-line"></i>tailnet boundary</span>
      </div>`
      : "";
    legend.innerHTML = `
      <div class="legend-row">
        <span class="legend-label">status</span>
        <span class="legend-item"><i class="legend-dot pulse" style="--c:#5ee7a5"></i>online</span>
        <span class="legend-item"><i class="legend-dot hollow" style="--c:#8494a8"></i>history</span>
        <span class="legend-item"><i class="legend-pin"></i>pinned</span>
      </div>
      <div class="legend-row">
        <span class="legend-label">device</span>
        ${kinds}
      </div>
      <div class="legend-row">
        <span class="legend-label">service</span>
        ${categories}
        ${tailnetOnly}
      </div>
      ${reach}
      <div class="legend-row">
        <span class="legend-label">traffic</span>
        <span class="legend-item"><i class="legend-flow" style="--c:${TRAFFIC.lan.color}"></i>LAN</span>
        <span class="legend-item"><i class="legend-flow" style="--c:${TRAFFIC.tailnet.color}"></i>tailnet</span>
        <span class="legend-item"><i class="legend-flow dashed" style="--c:#c9d7e6"></i>connections only</span>
      </div>
    `;
  }

  // The panel covers part of the canvas. Keep the selected bubble and its
  // label inside the map area that remains visible.
  function uncoveredMapRect() {
    const rect = {
      left: 12,
      top: size.chromeTop + 12,
      right: size.width - 12,
      bottom: size.height - 12,
    };
    if (!panel.classList.contains("open")) return rect;
    const box = panel.getBoundingClientRect();
    if (box.width >= size.width * 0.8) {
      rect.bottom = Math.min(rect.bottom, box.top - 12);
    } else {
      rect.right = Math.min(rect.right, box.left - 12);
    }
    return rect;
  }

  function focusTarget(x, y, zoom) {
    const rect = uncoveredMapRect();
    const focusX = (rect.left + rect.right) / 2;
    const focusY = rect.bottom < size.height - 12 ? (rect.top + rect.bottom) / 2 : size.height / 2;
    const viewportX = (size.width - size.insetRight) / 2;
    const viewportY = size.height / 2;
    return {
      x: x - (focusX - viewportX) / zoom,
      y: y - (focusY - viewportY) / zoom,
      zoom,
    };
  }

  function ensureSelectedVisible() {
    if (!view.selectedDeviceId) return;
    const node = view.nodes.get(view.selectedDeviceId);
    if (!node) return;
    const rect = uncoveredMapRect();
    const camera = view.target || view.camera;
    const zoom = camera.zoom;
    const point = {
      x: (node.x - camera.x) * zoom + (size.width - size.insetRight) / 2,
      y: (node.y - camera.y) * zoom + size.height / 2,
    };
    const radius = node.baseR * zoom;
    const fontSize = Math.max(9.5, 13 * zoom);
    const subSize = Math.max(8.5, 10.5 * zoom);
    const device = node.data;
    const reach = deviceReach(device);
    ctx.save();
    ctx.font = `600 ${fontSize}px ${FONT}`;
    const labelWidth = ctx.measureText(truncate(displayName(device), 24)).width;
    ctx.restore();
    const labelBottom = radius + 8 * zoom + 3 + fontSize +
      (device.online ? 0 : subSize + 3) +
      (view.tailnetPresent && hasTailnet(device) && (reach === "tailnet" || device.self) ? subSize + 3 : 0) +
      (zoom > 1.25 && reach !== "tailnet" ? subSize + 3 : 0) + 10;
    const halfWidth = Math.min(
      Math.max(radius + 10, labelWidth / 2 + 10),
      Math.max(1, (rect.right - rect.left) / 2),
    );
    const minX = rect.left + halfWidth;
    const maxX = rect.right - halfWidth;
    const minY = rect.top + radius + 10;
    const maxY = rect.bottom - labelBottom;
    let dx = 0;
    let dy = 0;
    if (point.x < minX) dx = minX - point.x;
    else if (point.x > maxX) dx = maxX - point.x;
    if (point.y < minY) dy = minY - point.y;
    else if (point.y > maxY) dy = maxY - point.y;
    if (Math.abs(dx) < 0.5 && Math.abs(dy) < 0.5) return;
    view.target = {
      x: camera.x - dx / zoom,
      y: camera.y - dy / zoom,
      zoom,
    };
    view.userAdjusted = true;
  }

  function setDockDetails(docked) {
    view.dockDetails = docked;
    try {
      window.localStorage.setItem("hearthlight.dock", docked ? "1" : "0");
    } catch (_error) {
      // Storage is optional.
    }
    const toggle = panel.querySelector(".panel-dock");
    if (toggle) {
      toggle.setAttribute("aria-pressed", String(docked));
      toggle.title = docked ? "Show details as an overlay" : "Dock details on wide screens";
    }
    resize();
    showToast(docked ? "Details docked" : "Details overlay");
  }

  function openPanel(deviceId) {
    view.selectedDeviceId = deviceId;
    renderPanel();
    syncPanelDock();
    panel.inert = false;
    panel.classList.add("open");
    panel.setAttribute("aria-hidden", "false");
    updateMapInset();
    ensureSelectedVisible();
  }

  function closePanel() {
    const hadFocus = panel.contains(document.activeElement);
    const returnFocus = view.panelReturnFocus;
    view.panelReturnFocus = null;
    view.selectedDeviceId = null;
    view.selectedServiceId = null;
    panel.inert = true;
    panel.classList.remove("open");
    panel.setAttribute("aria-hidden", "true");
    updateMapInset();
    if (!hadFocus) return;
    if (returnFocus) returnFocus.focus();
    else if (panel.contains(document.activeElement)) document.activeElement.blur();
  }

  function selectedDevice() {
    const snapshot = view.snapshot;
    if (!snapshot || !view.selectedDeviceId) return null;
    return snapshot.devices.find((candidate) => candidate.id === view.selectedDeviceId) || null;
  }

  function renderPanel() {
    const device = selectedDevice();
    if (!device) return;
    const name = displayName(device);
    const hostname = String(device.hostname || "").trim();
    const kind = KIND[device.kind] || KIND.device;
    const reach = deviceReach(device);
    const services = [...(device.services || [])].sort(
      (a, b) => Number(b.online) - Number(a.online) || a.port - b.port,
    );
    const onlineCount = services.filter((service) => service.online).length;
    const rows = services
      .map((service) => {
        const color = CATEGORY[service.category] || CATEGORY.other;
        const name = service.product || service.name || `TCP ${service.port}`;
        const sub = service.product && service.name && service.product !== service.name
          ? service.name
          : (service.protocol || "tcp").toUpperCase();
        const selected = service.id === view.selectedServiceId;
        const serviceReach = service.tailnet && service.local ? "both" : service.tailnet ? "tailnet" : "local";
        const reachTag = view.tailnetPresent
          ? `<span class="svc-reach ${serviceReach}" title="${escapeHTML(reachLabel(serviceReach))}">${reachShort(serviceReach)}</span>`
          : "";
        const state = serviceState(service);
        return `
          <li class="service-row ${service.online ? "on" : "off"} ${selected ? "selected" : ""}" data-service-id="${escapeHTML(service.id)}">
            <span class="svc-glyph" style="--c:${color}" aria-hidden="true">${glyphSVG(serviceGlyph(service))}</span>
            <span class="svc-main">
              <strong>${escapeHTML(name)}</strong>
              <span class="svc-sub">${escapeHTML(sub)}<span class="svc-state ${state.tone}" title="${escapeHTML(state.title)}" aria-label="${escapeHTML(state.title)}">${escapeHTML(state.label)}</span></span>
            </span>
            ${reachTag}
            <span class="svc-port">:${service.port}</span>
          </li>
        `;
      })
      .join("");

    const meta = [];
    if (view.tailnetPresent) {
      meta.push(`<div><dt>Reach</dt><dd>${escapeHTML(reachLabel(reach))}</dd></div>`);
    }
    if (hostname && hostname !== name) {
      meta.push(`<div><dt>Hostname</dt><dd>${escapeHTML(hostname)}</dd></div>`);
    }
    if (device.tailnetIp) {
      meta.push(`<div><dt>Tailnet IP</dt><dd>${escapeHTML(device.tailnetIp)}</dd></div>`);
    }
    if (device.tailnetName) {
      meta.push(`<div><dt>MagicDNS</dt><dd>${escapeHTML(device.tailnetName)}</dd></div>`);
    }
    if (device.tailnetOs) {
      meta.push(`<div><dt>Tailnet OS</dt><dd>${escapeHTML(device.tailnetOs)}</dd></div>`);
    }
    if (device.tailnetTags && device.tailnetTags.length) {
      meta.push(`<div><dt>Tags</dt><dd>${escapeHTML(device.tailnetTags.join(", "))}</dd></div>`);
    }
    meta.push(`<div><dt>Vendor</dt><dd>${escapeHTML(device.vendor || "unknown")}</dd></div>`);
    meta.push(`<div><dt>MAC</dt><dd>${escapeHTML(device.mac || "unknown")}</dd></div>`);
    meta.push(`<div><dt>First seen</dt><dd>${escapeHTML(relativeTime(device.firstSeen))}</dd></div>`);
    meta.push(`<div><dt>Last seen</dt><dd>${escapeHTML(relativeTime(device.lastSeen))}</dd></div>`);

    panel.innerHTML = `
      <div class="panel-head">
        <div class="panel-kicker">
          <span class="status-dot ${device.online ? "on" : "off"}"></span>
          <canvas class="panel-kind" aria-hidden="true"></canvas>
          ${device.online ? "online" : "history"} · <span class="panel-kind-label" style="--c:${kind.color}">${escapeHTML(kind.label)}</span>${view.tailnetPresent ? ` · ${escapeHTML(reachLabel(reach))}` : ""}
        </div>
        <div class="panel-tools">
          <button class="panel-dock" type="button" aria-pressed="${view.dockDetails}" title="${view.dockDetails ? "Show details as an overlay" : "Dock details on wide screens"}">
            <span class="panel-dock-icon" aria-hidden="true"></span>dock details
          </button>
          <button class="panel-close" type="button" aria-label="Close details">×</button>
        </div>
      </div>
      <h2>${escapeHTML(name)}</h2>
      <p class="ip">${escapeHTML(device.ip)}${device.self ? " · this host" : ""}</p>
      <dl class="meta panel-traffic" hidden></dl>
      <dl class="meta">
        ${meta.join("")}
      </dl>
      <h3>Services <span>${onlineCount}/${services.length} up</span></h3>
      ${rows || '<div class="empty-services">No TCP services found on this device.</div>'}
    `;
    paintPanelKind(device);
    renderPanelTraffic();
  }

  function paintPanelKind(device) {
    const node = panel.querySelector(".panel-kind");
    if (!node) return;
    const kind = KIND[device.kind] || KIND.device;
    const box = 18;
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    node.width = Math.round(box * dpr);
    node.height = Math.round(box * dpr);
    const icon = node.getContext("2d");
    icon.setTransform(dpr, 0, 0, dpr, 0, 0);
    icon.clearRect(0, 0, box, box);
    drawDeviceIcon(device.kind, box / 2, box / 2, box * 0.94, kind.color, icon);
  }

  // Keep relative times current without rebuilding the panel and losing scroll.
  function refreshServiceTimes() {
    const device = selectedDevice();
    if (!device || !panel.classList.contains("open")) return;
    const services = device.services || [];
    for (const row of panel.querySelectorAll(".service-row")) {
      const service = services.find((candidate) => candidate.id === row.dataset.serviceId);
      const node = row.querySelector(".svc-state");
      if (!service || !node) continue;
      const state = serviceState(service);
      node.textContent = state.label;
      node.title = state.title;
      node.setAttribute("aria-label", state.title);
      node.classList.toggle("on", state.tone === "on");
      node.classList.toggle("off", state.tone === "off");
    }
  }

  function selectNode(node, returnFocus = null) {
    if (!node) return;
    hideTooltip();
    view.panelReturnFocus = returnFocus;
    if (node.type === "device") {
      view.selectedServiceId = null;
      openPanel(node.id);
    } else {
      view.selectedServiceId = node.id;
      openPanel(node.parentId);
    }
  }

  function showTooltip(node, x, y) {
    if (node.type === "device") {
      const device = node.data;
      const name = displayName(device);
      const hostname = String(device.hostname || "").trim();
      const services = device.services || [];
      const online = services.filter((service) => service.online).length;
      const kind = KIND[device.kind] || KIND.device;
      const reach = deviceReach(device);
      const fullHostname = hostname && hostname !== name
        ? `<div class="tip-meta">${escapeHTML(hostname)}</div>`
        : "";
      const tailnet = device.tailnetIp
        ? `<div class="tip-meta">${escapeHTML(device.tailnetIp)}${device.tailnetOs ? ` · ${escapeHTML(device.tailnetOs)}` : ""}${device.tailnetName ? ` · ${escapeHTML(device.tailnetName)}` : ""}</div>`
        : "";
      tooltip.innerHTML = `
        <strong>${escapeHTML(name)}</strong>
        <div>${escapeHTML(device.ip)} · ${escapeHTML(kind.label)}${device.online ? "" : " · history"}</div>
        ${fullHostname}
        <div class="tip-meta">${view.tailnetPresent ? `${escapeHTML(reachLabel(reach))} · ` : ""}${online}/${services.length} services up${device.pinned ? " · pinned" : ""}${device.vendor ? ` · ${escapeHTML(device.vendor)}` : ""}</div>
        ${tailnet}
      `;
    } else {
      const parent = view.nodes.get(node.parentId);
      const serviceReach = node.data.tailnet && node.data.local ? "both" : node.data.tailnet ? "tailnet" : "local";
      tooltip.innerHTML = `
        <strong>${escapeHTML(node.data.product || node.data.name || "service")}</strong>
        <div>${escapeHTML(node.data.name || "TCP")} · port ${node.data.port}</div>
        <div class="tip-meta">${parent ? `${escapeHTML(displayName(parent.data))} · ` : ""}${node.data.online ? "online" : "history"}${view.tailnetPresent ? ` · ${escapeHTML(reachLabel(serviceReach))}` : ""}</div>
      `;
    }
    tooltip.classList.add("show");
    positionTooltip(x, y);
  }

  function positionTooltip(x, y) {
    const rect = tooltip.getBoundingClientRect();
    let left = x + 16;
    let top = y + 16;
    if (left + rect.width > size.width - 12) left = x - rect.width - 16;
    if (top + rect.height > size.height - 12) top = y - rect.height - 16;
    tooltip.style.left = `${Math.max(8, left)}px`;
    tooltip.style.top = `${Math.max(8, top)}px`;
  }

  function hideTooltip() {
    tooltip.classList.remove("show");
  }

  function showToast(message) {
    toast.textContent = message;
    toast.classList.add("show");
    window.clearTimeout(view.toastTimer);
    view.toastTimer = window.setTimeout(() => toast.classList.remove("show"), 3400);
  }

  function setHistoryVisible(show, announce = true) {
    view.showOffline = show;
    historyBtn.setAttribute("aria-pressed", String(show));
    syncSettingsControls();
    try {
      window.localStorage.setItem("hearthlight.history", show ? "1" : "0");
    } catch (_error) {
      // Storage is optional.
    }
    if (announce) showToast(show ? "History shown" : "History hidden");
    hideTooltip();
  }

  function setTailnetVisible(show, announce = true) {
    view.showTailnet = show;
    if (tailnetBtn) tailnetBtn.setAttribute("aria-pressed", String(show));
    syncSettingsControls();
    try {
      window.localStorage.setItem("hearthlight.tailnet", show ? "1" : "0");
    } catch (_error) {
      // Storage is optional.
    }
    view.territorySig = "";
    if (announce) showToast(show ? "Tailnet shown" : "Tailnet hidden");
    hideTooltip();
    syncChrome();
  }

  // Native modal dialogs trap focus, but an explicit loop keeps Tab inside the
  // dialog in every browser and makes the boundary behaviour testable.
  const FOCUSABLE_SELECTOR = 'button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

  function trapSettingsFocus(event) {
    if (event.key !== "Tab") return;
    const focusable = [...settingsDialog.querySelectorAll(FOCUSABLE_SELECTOR)]
      .filter((element) => element.getClientRects().length > 0);
    if (!focusable.length) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function trapDeviceSheetFocus(event) {
    if (event.key !== "Tab") return;
    const focusable = [...deviceList.querySelectorAll(FOCUSABLE_SELECTOR)]
      .filter((element) => element.getClientRects().length > 0);
    if (!focusable.length) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function openSettings() {
    if (!settingsDialog || settingsDialog.open) return;
    syncSettingsControls();
    hideTooltip();
    settingsDialog.showModal();
    settingsBtn.setAttribute("aria-expanded", "true");
    settingsClose.focus();
  }

  function closeSettings() {
    if (!settingsDialog || !settingsDialog.open) return;
    settingsDialog.close();
    settingsBtn.setAttribute("aria-expanded", "false");
    settingsBtn.focus();
  }

  function hitTest(screenX, screenY) {
    const zoom = view.camera.zoom;
    let closestService = null;
    let serviceDistance = Infinity;
    for (const node of view.nodes.values()) {
      if (node.type !== "service" || serviceAlpha(node, zoom) < 0.15) continue;
      const parent = view.nodes.get(node.parentId);
      if (!parent || !deviceVisible(parent.data)) continue;
      const point = worldToScreen(node.x, node.y);
      const dx = screenX - point.x;
      const dy = screenY - point.y;
      const radius = Math.max(10, SERVICE_R * zoom * (1 + node.hover * 0.35)) + 3;
      const distance = dx * dx + dy * dy;
      if (distance <= radius * radius && distance < serviceDistance) {
        serviceDistance = distance;
        closestService = node;
      }
    }
    if (closestService) return closestService;

    let closestDevice = null;
    let deviceDistance = Infinity;
    for (const node of view.nodes.values()) {
      if (node.type !== "device") continue;
      if (!deviceVisible(node.data)) continue;
      const point = worldToScreen(node.x, node.y);
      const dx = screenX - point.x;
      const dy = screenY - point.y;
      const radius = node.baseR * zoom * (1 + node.hover * 0.06);
      const distance = dx * dx + dy * dy;
      if (distance <= radius * radius && distance < deviceDistance) {
        deviceDistance = distance;
        closestDevice = node;
      }
    }
    return closestDevice;
  }

  function updateHover(screenX, screenY) {
    if (view.dragging) {
      hideTooltip();
      return;
    }
    const node = hitTest(screenX, screenY);
    view.hoverId = node ? node.id : null;
    canvas.style.cursor = node ? "pointer" : "grab";
    if (!node) {
      hideTooltip();
      return;
    }
    showTooltip(node, screenX, screenY);
  }

  function drawBackground() {
    const background = ctx.createRadialGradient(
      size.width * 0.5,
      size.height * 0.38,
      0,
      size.width * 0.5,
      size.height * 0.5,
      Math.max(size.width, size.height) * 0.85,
    );
    background.addColorStop(0, "#0a1526");
    background.addColorStop(0.55, "#060b14");
    background.addColorStop(1, "#03060c");
    ctx.fillStyle = background;
    ctx.fillRect(0, 0, size.width, size.height);

    const gap = 46;
    const offsetX = ((-view.camera.x * view.camera.zoom * 0.08) % gap + gap) % gap;
    const offsetY = ((-view.camera.y * view.camera.zoom * 0.08) % gap + gap) % gap;
    ctx.fillStyle = "rgba(120, 185, 230, 0.055)";
    for (let x = offsetX; x < size.width; x += gap) {
      for (let y = offsetY; y < size.height; y += gap) {
        ctx.fillRect(x, y, 1, 1);
      }
    }

    const glow = ctx.createRadialGradient(
      size.width * 0.72,
      -size.height * 0.2,
      0,
      size.width * 0.72,
      -size.height * 0.2,
      size.height * 0.95,
    );
    glow.addColorStop(0, "rgba(70, 150, 190, 0.1)");
    glow.addColorStop(1, "rgba(70, 150, 190, 0)");
    ctx.fillStyle = glow;
    ctx.fillRect(0, 0, size.width, size.height);
  }

  function roundRectPath(x, y, width, height, radius, target = ctx) {
    const r = Math.min(radius, Math.abs(width) / 2, Math.abs(height) / 2);
    target.beginPath();
    target.moveTo(x + r, y);
    target.arcTo(x + width, y, x + width, y + height, r);
    target.arcTo(x + width, y + height, x, y + height, r);
    target.arcTo(x, y + height, x, y, r);
    target.arcTo(x, y, x + width, y, r);
    target.closePath();
  }

  function drawDeviceIcon(kind, x, y, iconSize, color, ctx) {
    const u = iconSize / 2;
    ctx.save();
    ctx.translate(x, y);
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.lineWidth = Math.max(1.2, iconSize * 0.075);
    ctx.lineCap = "round";
    ctx.lineJoin = "round";

    switch (kind) {
      case "router":
        ctx.beginPath();
        ctx.arc(0, u * 0.45, u * 0.9, Math.PI * 1.18, Math.PI * 1.82);
        ctx.stroke();
        ctx.beginPath();
        ctx.arc(0, u * 0.45, u * 0.55, Math.PI * 1.18, Math.PI * 1.82);
        ctx.stroke();
        ctx.beginPath();
        ctx.arc(0, u * 0.45, u * 0.14, 0, TAU);
        ctx.fill();
        break;
      case "server":
        for (let row = -1; row <= 1; row += 1) {
          roundRectPath(-u * 0.75, row * u * 0.52 - u * 0.19, u * 1.5, u * 0.36, u * 0.08, ctx);
          ctx.stroke();
        }
        ctx.beginPath();
        ctx.arc(u * 0.45, -u * 0.01, u * 0.07, 0, TAU);
        ctx.fill();
        break;
      case "computer":
        roundRectPath(-u * 0.9, -u * 0.72, u * 1.8, u * 1.16, u * 0.12, ctx);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(-u * 0.38, u * 0.62);
        ctx.lineTo(u * 0.38, u * 0.62);
        ctx.moveTo(0, u * 0.44);
        ctx.lineTo(0, u * 0.62);
        ctx.stroke();
        break;
      case "phone":
        roundRectPath(-u * 0.36, -u * 0.8, u * 0.72, u * 1.6, u * 0.16, ctx);
        ctx.stroke();
        ctx.beginPath();
        ctx.arc(0, u * 0.56, u * 0.06, 0, TAU);
        ctx.fill();
        break;
      case "nas":
        ctx.beginPath();
        ctx.ellipse(0, -u * 0.42, u * 0.65, u * 0.22, 0, 0, TAU);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(-u * 0.65, -u * 0.42);
        ctx.lineTo(-u * 0.65, u * 0.35);
        ctx.moveTo(u * 0.65, -u * 0.42);
        ctx.lineTo(u * 0.65, u * 0.35);
        ctx.stroke();
        ctx.beginPath();
        ctx.ellipse(0, u * 0.35, u * 0.65, u * 0.22, 0, 0, Math.PI);
        ctx.stroke();
        break;
      case "printer":
        roundRectPath(-u * 0.44, -u * 0.86, u * 0.88, u * 0.58, u * 0.06, ctx);
        ctx.stroke();
        roundRectPath(-u * 0.85, -u * 0.26, u * 1.7, u * 0.68, u * 0.12, ctx);
        ctx.stroke();
        roundRectPath(-u * 0.44, u * 0.16, u * 0.88, u * 0.58, u * 0.05, ctx);
        ctx.stroke();
        break;
      case "media":
        ctx.beginPath();
        ctx.arc(0, 0, u * 0.78, 0, TAU);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(-u * 0.2, -u * 0.32);
        ctx.lineTo(u * 0.38, 0);
        ctx.lineTo(-u * 0.2, u * 0.32);
        ctx.closePath();
        ctx.fill();
        break;
      case "iot":
        roundRectPath(-u * 0.5, -u * 0.5, u, u, u * 0.15, ctx);
        ctx.stroke();
        for (let i = -1; i <= 1; i += 1) {
          ctx.beginPath();
          ctx.moveTo(i * u * 0.34, -u * 0.5);
          ctx.lineTo(i * u * 0.34, -u * 0.78);
          ctx.moveTo(i * u * 0.34, u * 0.5);
          ctx.lineTo(i * u * 0.34, u * 0.78);
          ctx.moveTo(-u * 0.5, i * u * 0.34);
          ctx.lineTo(-u * 0.78, i * u * 0.34);
          ctx.moveTo(u * 0.5, i * u * 0.34);
          ctx.lineTo(u * 0.78, i * u * 0.34);
          ctx.stroke();
        }
        ctx.beginPath();
        ctx.arc(0, 0, u * 0.12, 0, TAU);
        ctx.fill();
        break;
      default: {
        const points = [
          [0, -u * 0.58],
          [-u * 0.56, u * 0.42],
          [u * 0.56, u * 0.42],
        ];
        ctx.beginPath();
        ctx.moveTo(points[0][0], points[0][1]);
        ctx.lineTo(points[1][0], points[1][1]);
        ctx.lineTo(points[2][0], points[2][1]);
        ctx.closePath();
        ctx.stroke();
        for (const [px, py] of points) {
          ctx.beginPath();
          ctx.arc(px, py, u * 0.13, 0, TAU);
          ctx.fill();
        }
        break;
      }
    }
    ctx.restore();
  }

  const glyphPathCache = new Map();
  const glyphPath = (data) => {
    let path = glyphPathCache.get(data);
    if (!path) {
      path = new Path2D(data);
      glyphPathCache.set(data, path);
    }
    return path;
  };

  function drawServiceGlyph(glyph, x, y, glyphSize, color) {
    const shapes = SERVICE_GLYPHS[glyph] || SERVICE_GLYPHS.dot;
    const scale = glyphSize / 8;
    ctx.save();
    ctx.translate(x, y);
    ctx.scale(scale, scale);
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.lineWidth = 1;
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    for (const [kind, ...values] of shapes) {
      switch (kind) {
        case "ring":
          ctx.beginPath();
          ctx.arc(values[0], values[1], values[2], 0, TAU);
          ctx.stroke();
          break;
        case "ellipse":
          ctx.beginPath();
          ctx.ellipse(values[0], values[1], values[2], values[3], 0, 0, TAU);
          ctx.stroke();
          break;
        case "dot":
          ctx.beginPath();
          ctx.arc(values[0], values[1], values[2], 0, TAU);
          ctx.fill();
          break;
        case "rect":
          roundRectPath(values[0], values[1], values[2], values[3], values[4]);
          ctx.stroke();
          break;
        case "fill":
          ctx.fill(glyphPath(values[0]));
          break;
        default:
          ctx.stroke(glyphPath(values[0]));
          break;
      }
    }
    ctx.restore();
  }

  function strokeLabel(text, x, y, fontSize) {
    ctx.lineWidth = Math.max(3, fontSize * 0.28);
    ctx.strokeStyle = "rgba(3, 7, 12, 0.86)";
    ctx.lineJoin = "round";
    ctx.strokeText(text, x, y);
  }

  // Pinned devices wear a warm ring, visible at every zoom, so manual
  // placement reads apart from the automatic layout and the history dash.
  function drawPinAffordance(point, radius, alpha) {
    const scale = clamp(view.camera.zoom, 0.6, 1.4);
    const offset = 3.4 + scale * 1.4;
    const headY = point.y - radius - offset;
    ctx.save();
    ctx.globalAlpha = alpha * 0.9;
    ctx.beginPath();
    ctx.arc(point.x, point.y, radius + offset, 0, TAU);
    ctx.strokeStyle = rgba("#ffd166", 0.64);
    ctx.lineWidth = Math.max(1, 1.1 * scale);
    ctx.stroke();
    ctx.beginPath();
    ctx.arc(point.x, headY, Math.max(1.5, 1.8 * scale), 0, TAU);
    ctx.fillStyle = "#ffd166";
    ctx.fill();
    ctx.restore();
  }

  function drawDevice(node, time, zoom) {
    const device = node.data;
    if (!deviceVisible(device)) return;
    const historicAlpha = deviceHistoricAlpha(node, zoom);
    if (historicAlpha <= 0.01) return;
    const point = worldToScreen(node.x, node.y);
    const radius = node.baseR * zoom * (1 + node.hover * 0.06);
    if (
      point.x < -radius * 3 ||
      point.x > size.width + radius * 3 ||
      point.y < -radius * 3 ||
      point.y > size.height + radius * 3
    ) {
      return;
    }

    const color = (KIND[device.kind] || KIND.device).color;
    const active = device.online;
    const reach = deviceReach(device);

    if (active) {
      const glow = ctx.createRadialGradient(
        point.x,
        point.y,
        radius * 0.65,
        point.x,
        point.y,
        radius * 2.6,
      );
      glow.addColorStop(0, rgba(color, 0.22));
      glow.addColorStop(0.55, rgba(color, 0.06));
      glow.addColorStop(1, rgba(color, 0));
      ctx.fillStyle = glow;
      ctx.beginPath();
      ctx.arc(point.x, point.y, radius * 2.6, 0, TAU);
      ctx.fill();

      if (!view.reducedMotion) {
        const phase = ((time / 2600) + node.seed) % 1;
        ctx.beginPath();
        ctx.arc(point.x, point.y, radius + phase * 24, 0, TAU);
        ctx.strokeStyle = rgba(color, (1 - phase) * 0.3);
        ctx.lineWidth = 1.2;
        ctx.stroke();
      }
    }

    ctx.beginPath();
    ctx.arc(point.x, point.y, radius, 0, TAU);
    ctx.fillStyle = "rgba(6, 12, 21, 0.94)";
    ctx.fill();
    const inner = ctx.createRadialGradient(
      point.x,
      point.y,
      radius * 0.1,
      point.x,
      point.y,
      radius,
    );
    inner.addColorStop(0, rgba(color, active ? 0.28 : 0.08));
    inner.addColorStop(1, rgba(color, 0.02));
    ctx.fillStyle = inner;
    ctx.fill();

    ctx.save();
    ctx.globalAlpha = historicAlpha;
    ctx.setLineDash(active ? [] : [4, 5]);
    ctx.lineWidth = active ? 1.6 : 1.1;
    ctx.strokeStyle = rgba(color, active ? 0.9 : 0.5);
    ctx.beginPath();
    ctx.arc(point.x, point.y, radius, 0, TAU);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.restore();

    if (device.pinned) drawPinAffordance(point, radius, historicAlpha);

    ctx.save();
    ctx.globalAlpha = historicAlpha;
    drawDeviceIcon(device.kind, point.x, point.y, radius * 1.52, rgba(color, active ? 0.92 : 0.55), ctx);
    ctx.restore();

    if (tailnetShown()) {
      if (device.self) {
        ctx.save();
        ctx.globalAlpha = historicAlpha * 0.9;
        ctx.beginPath();
        ctx.arc(point.x, point.y, radius + 6.5, 0, TAU);
        ctx.setLineDash([2, 4]);
        ctx.strokeStyle = "rgba(255, 255, 255, 0.38)";
        ctx.lineWidth = 1;
        ctx.stroke();
        ctx.restore();
      }
      drawReachBadge(point, radius, reach, historicAlpha);
    }

    if (view.selectedDeviceId === node.id || view.hoverId === node.id) {
      ctx.beginPath();
      ctx.arc(point.x, point.y, radius + 5, 0, TAU);
      ctx.strokeStyle = rgba("#ffffff", view.selectedDeviceId === node.id ? 0.45 : 0.25);
      ctx.lineWidth = 1.1;
      ctx.stroke();
    }

    const serviceCount = (device.services || []).length;
    if (serviceCount > 0 && zoom < 0.92) {
      const onlineCount = (device.services || []).filter((service) => service.online).length;
      const badgeX = point.x + radius * 0.72;
      const badgeY = point.y - radius * 0.72;
      const badgeRadius = Math.max(7, 7.5 * clamp(zoom + 0.45, 0.8, 1.2));
      ctx.save();
      ctx.globalAlpha = historicAlpha;
      ctx.beginPath();
      ctx.arc(badgeX, badgeY, badgeRadius, 0, TAU);
      ctx.fillStyle = "rgba(6, 12, 21, 0.95)";
      ctx.fill();
      ctx.strokeStyle = rgba(onlineCount > 0 ? color : "#718198", 0.85);
      ctx.lineWidth = 1;
      ctx.stroke();
      ctx.fillStyle = onlineCount > 0 ? "#dcecf8" : "#7f8fa6";
      ctx.font = `600 ${Math.max(7.5, badgeRadius * 1.05)}px ${MONO}`;
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(String(serviceCount), badgeX, badgeY + 0.5);
      ctx.restore();
    }

    const forced = view.hoverId === node.id || view.selectedDeviceId === node.id;
    const labelAlpha = forced ? 1 : smoothstep(0.26, 0.52, zoom);
    if (labelAlpha <= 0.01) return;

    const name = truncate(displayName(device), 24);
    const fontSize = Math.max(9.5, 13 * zoom);
    ctx.save();
    ctx.globalAlpha = labelAlpha * historicAlpha;
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    ctx.font = `600 ${fontSize}px ${FONT}`;
    const textY = point.y + radius + 8 * zoom + 3;
    strokeLabel(name, point.x, textY, fontSize);
    ctx.fillStyle = active ? (reach === "tailnet" ? "#d8f8ff" : "#eaf4fd") : "#8b9aad";
    ctx.fillText(name, point.x, textY);

    const subline = view.tailnetPresent && hasTailnet(device) && (reach === "tailnet" || device.self)
      ? (device.self ? `this host · ${device.tailnetIp || device.ip}` : (device.tailnetIp || device.ip))
      : "";
    let subY = textY + fontSize + 3;
    const subSize = Math.max(8.5, 10.5 * zoom);
    if (!active) {
      ctx.globalAlpha = labelAlpha * historicAlpha;
      ctx.font = `500 ${subSize}px ${MONO}`;
      ctx.fillStyle = "#7f8fa6";
      const lastSeen = lastSeenLabel(device);
      strokeLabel(lastSeen, point.x, subY, subSize);
      ctx.fillText(lastSeen, point.x, subY);
      subY += subSize + 3;
    }
    if (subline) {
      ctx.globalAlpha = labelAlpha * historicAlpha * smoothstep(0.4, 0.62, zoom);
      ctx.font = `500 ${subSize}px ${MONO}`;
      ctx.fillStyle = device.self ? "#9bb0c5" : "#7abccc";
      strokeLabel(subline, point.x, subY, subSize);
      ctx.fillText(subline, point.x, subY);
      subY += subSize + 3;
    }

    if (zoom > 1.25) {
      ctx.globalAlpha = labelAlpha * historicAlpha * smoothstep(1.25, 1.6, zoom);
      ctx.font = `500 ${subSize}px ${MONO}`;
      ctx.fillStyle = "#73869c";
      if (reach !== "tailnet") {
        strokeLabel(device.ip, point.x, subY, subSize);
        ctx.fillText(device.ip, point.x, subY);
      }
    }
    ctx.restore();
  }

  function drawService(node, zoom) {
    const alpha = serviceAlpha(node, zoom);
    if (alpha <= 0.015) return;
    const parentNode = view.nodes.get(node.parentId);
    if (!parentNode || !deviceVisible(parentNode.data)) return;
    const point = worldToScreen(node.x, node.y);
    const radius = Math.max(3, SERVICE_R * zoom * (1 + node.hover * 0.35));
    if (
      point.x < -radius * 4 ||
      point.x > size.width + radius * 4 ||
      point.y < -radius * 4 ||
      point.y > size.height + radius * 4
    ) {
      return;
    }

    const color = CATEGORY[node.data.category] || CATEGORY.other;
    const active = node.data.online;

    ctx.save();
    ctx.globalAlpha = alpha;
    if (active) {
      const glow = ctx.createRadialGradient(
        point.x,
        point.y,
        radius * 0.4,
        point.x,
        point.y,
        radius * 3,
      );
      glow.addColorStop(0, rgba(color, 0.16));
      glow.addColorStop(1, rgba(color, 0));
      ctx.fillStyle = glow;
      ctx.beginPath();
      ctx.arc(point.x, point.y, radius * 3, 0, TAU);
      ctx.fill();

      ctx.beginPath();
      ctx.arc(point.x, point.y, radius, 0, TAU);
      ctx.fillStyle = rgba(color, 0.24);
      ctx.fill();
      ctx.strokeStyle = rgba(color, 0.9);
      ctx.lineWidth = 1.1;
      ctx.stroke();

      if (zoom >= SERVICE_GLYPH_ZOOM) {
        drawServiceGlyph(node.glyph || serviceGlyph(node.data), point.x, point.y, radius * 1.6, rgba(color, 0.95));
      } else {
        ctx.beginPath();
        ctx.arc(point.x, point.y, Math.max(1.2, radius * 0.28), 0, TAU);
        ctx.fillStyle = rgba(color, 0.95);
        ctx.fill();
      }
    } else {
      ctx.setLineDash([2.5, 3]);
      ctx.beginPath();
      ctx.arc(point.x, point.y, radius, 0, TAU);
      ctx.strokeStyle = rgba(color, 0.55);
      ctx.lineWidth = 1;
      ctx.stroke();
      ctx.setLineDash([]);
      if (zoom >= SERVICE_GLYPH_ZOOM) {
        drawServiceGlyph(node.glyph || serviceGlyph(node.data), point.x, point.y, radius * 1.6, rgba(color, 0.62));
      }
    }
    if (tailnetShown() && node.data.tailnet && !node.data.local) {
      ctx.beginPath();
      ctx.arc(point.x, point.y, radius + 4.4, 0, TAU);
      ctx.setLineDash([1.5, 3.2]);
      ctx.strokeStyle = "rgba(94, 231, 255, 0.55)";
      ctx.lineWidth = 0.9;
      ctx.stroke();
      ctx.setLineDash([]);
    }
    ctx.restore();

    if (view.hoverId === node.id || view.selectedServiceId === node.id) {
      ctx.beginPath();
      ctx.arc(point.x, point.y, radius + 4, 0, TAU);
      ctx.strokeStyle = "rgba(255, 255, 255, 0.5)";
      ctx.lineWidth = 1;
      ctx.stroke();
    }

    const forced = view.hoverId === node.id || view.selectedServiceId === node.id;
    const labelAlpha = forced ? 1 : serviceLabelAlpha(node, zoom);
    if (labelAlpha <= 0.02) return;

    const label = truncate(node.data.product || node.data.name || `port ${node.data.port}`, 26);
    const fontSize = Math.max(8.5, 10.5 * zoom);
    const parent = view.nodes.get(node.parentId);
    let textX = point.x + radius + 6 * zoom + 2;
    let textY = point.y + fontSize * 0.35;
    let align = "left";
    if (parent) {
      const dx = node.x - parent.x;
      const dy = node.y - parent.y;
      const distance = Math.hypot(dx, dy) || 1;
      const ux = dx / distance;
      const uy = dy / distance;
      if (Math.abs(ux) < 0.35) {
        align = "center";
        textX = point.x;
        textY = point.y + (uy < 0 ? -(radius + fontSize * 0.85) : radius + fontSize * 0.85);
      } else if (ux < 0) {
        align = "right";
        textX = point.x - radius - 6 * zoom - 2;
      } else {
        align = "left";
        textX = point.x + radius + 6 * zoom + 2;
      }
    }
    ctx.save();
    ctx.textAlign = align;
    ctx.textBaseline = "middle";
    ctx.font = `500 ${fontSize}px ${FONT}`;
    const labelWidth = ctx.measureText(label).width;
    const labelLeft = align === "center" ? textX - labelWidth / 2 : align === "right" ? textX - labelWidth : textX;
    const labelRect = { left: labelLeft, top: textY - fontSize * 0.7, width: labelWidth, height: fontSize * 1.4 };
    const blocked = !forced && (
      view.deviceLabelRects.some((rect) => rectsOverlap(rect, labelRect, 4)) ||
      view.serviceLabelRects.some((rect) => rectsOverlap(rect, labelRect, 4))
    );
    if (blocked) {
      ctx.restore();
      return;
    }
    view.serviceLabelRects.push(labelRect);
    ctx.globalAlpha = labelAlpha * alpha;
    strokeLabel(label, textX, textY, fontSize);
    ctx.fillStyle = active
      ? (node.data.tailnet && !node.data.local ? "#a9e7f1" : "#d8e8f6")
      : "#8191a5";
    ctx.fillText(label, textX, textY);

    if (zoom > 1.45) {
      ctx.globalAlpha = labelAlpha * alpha * smoothstep(1.45, 1.8, zoom);
      ctx.font = `500 ${Math.max(8, 9.5 * zoom)}px ${MONO}`;
      ctx.fillStyle = "#6f8298";
      const portY = textY + fontSize * 0.95;
      const portText = `:${node.data.port}`;
      strokeLabel(portText, textX, portY, Math.max(8, 9.5 * zoom));
      ctx.fillText(portText, textX, portY);
    }
    ctx.restore();
  }

  function draw(time) {
    ctx.setTransform(size.dpr, 0, 0, size.dpr, 0, 0);
    ctx.clearRect(0, 0, size.width, size.height);
    drawBackground();

    view.deviceLabelRects = [];
    view.serviceLabelRects = [];
    if (tailnetShown() && view.territories) {
      const layer = renderTerritoryLayer();
      ctx.save();
      ctx.globalCompositeOperation = "screen";
      ctx.drawImage(layer, 0, 0, size.width, size.height);
      ctx.restore();
      drawTerritoryBoundaries();
      drawRegionLabels();
    }

    const zoom = view.camera.zoom;
    for (const node of view.nodes.values()) {
      if (node.type !== "device") continue;
      const rect = deviceLabelRect(node, zoom);
      if (rect) view.deviceLabelRects.push(rect);
    }
    drawTraffic(time, zoom);
    for (const link of view.links) {
      const service = view.nodes.get(link.serviceId);
      const device = view.nodes.get(link.deviceId);
      if (!service || !device) continue;
      if (!deviceVisible(device.data)) continue;
      const alpha = serviceAlpha(service, zoom);
      if (alpha <= 0.01) continue;
      if (!device.data.online && !view.showOffline) continue;
      const related = view.hoverId === service.id ||
        view.hoverId === device.id ||
        view.selectedDeviceId === device.id;
      const color = CATEGORY[service.data.category] || CATEGORY.other;
      const from = worldToScreen(device.x, device.y);
      const to = worldToScreen(service.x, service.y);
      ctx.beginPath();
      ctx.moveTo(from.x, from.y);
      ctx.lineTo(to.x, to.y);
      ctx.strokeStyle = rgba(color, (related ? 0.32 : 0.1) * alpha * (service.data.online ? 1 : 0.6));
      ctx.lineWidth = 1;
      ctx.stroke();
    }

    for (const node of view.nodes.values()) {
      if (node.type === "service") drawService(node, zoom);
    }
    for (const node of view.nodes.values()) {
      if (node.type === "device") drawDevice(node, time, zoom);
    }
  }

  const trafficKey = (flow) => `${flow.deviceId}|${flow.via}`;

  function formatRate(bytesPerSecond) {
    const value = Math.max(0, bytesPerSecond || 0);
    if (value < 1024) return `${Math.round(value)} B/s`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(value < 10240 ? 1 : 0)} KB/s`;
    return `${(value / 1024 / 1024).toFixed(1)} MB/s`;
  }

  function applyTraffic(sample) {
    if (!sample || !Array.isArray(sample.flows)) return;
    const present = new Set();
    for (const flow of sample.flows) {
      if (!TRAFFIC[flow.via]) continue;
      const key = trafficKey(flow);
      present.add(key);
      let entry = view.traffic.get(key);
      if (!entry) {
        entry = {
          key,
          deviceId: flow.deviceId,
          via: flow.via,
          rxShown: 0,
          txShown: 0,
          alpha: 0,
          phase: (hashString(key) % 1000) / 1000,
        };
        view.traffic.set(key, entry);
      }
      entry.conns = flow.conns || 0;
      entry.measured = Boolean(flow.measured);
      entry.rx = flow.rxBps || 0;
      entry.tx = flow.txBps || 0;
      entry.active = true;
    }
    for (const [key, entry] of view.traffic) {
      if (!present.has(key)) {
        entry.active = false;
        entry.rx = 0;
        entry.tx = 0;
      }
    }
    renderPanelTraffic();
  }

  function updateTraffic(dt) {
    const fade = Math.min(1, dt * 3);
    const ease = Math.min(1, dt * 2.5);
    for (const [key, entry] of view.traffic) {
      entry.alpha += ((entry.active ? 1 : 0) - entry.alpha) * fade;
      entry.rxShown += (entry.rx - entry.rxShown) * ease;
      entry.txShown += (entry.tx - entry.txShown) * ease;
      if (!entry.active && entry.alpha < 0.01) view.traffic.delete(key);
    }
  }

  function selfNode() {
    for (const node of view.nodes.values()) {
      if (node.type === "device" && node.data.self) return node;
    }
    return null;
  }

  // Log-scaled weight: ~1 at 1 KB/s, ~3 at 1 MB/s.
  const rateWeight = (bytesPerSecond) => Math.log10(1 + bytesPerSecond / 1024);

  function bezierPoint(p0, c, p1, t) {
    const u = 1 - t;
    return {
      x: u * u * p0.x + 2 * u * t * c.x + t * t * p1.x,
      y: u * u * p0.y + 2 * u * t * c.y + t * t * p1.y,
    };
  }

  function drawTraffic(time, zoom) {
    if (!view.traffic.size) return;
    const self = selfNode();
    if (!self || !deviceVisible(self.data)) return;
    const from = worldToScreen(self.x, self.y);
    const scale = clamp(zoom, 0.6, 1.4);
    const seconds = time / 1000;

    ctx.save();
    ctx.lineCap = "round";
    for (const entry of view.traffic.values()) {
      const node = view.nodes.get(entry.deviceId);
      if (!node || node === self || !node.data.online || !deviceVisible(node.data)) continue;
      if (entry.via === "tailnet" && !view.showTailnet) continue;
      const style = TRAFFIC[entry.via];
      const to = worldToScreen(node.x, node.y);
      const dx = to.x - from.x;
      const dy = to.y - from.y;
      const distance = Math.hypot(dx, dy);
      if (distance < 1) continue;

      // Bow LAN and tailnet edges to opposite sides so a dual host shows both.
      const bend = distance * 0.16 * style.bend;
      const control = {
        x: (from.x + to.x) / 2 - (dy / distance) * bend,
        y: (from.y + to.y) / 2 + (dx / distance) * bend,
      };
      const related = view.hoverId === node.id || view.selectedDeviceId === node.id ||
        view.hoverId === self.id || view.selectedDeviceId === self.id;
      const emphasis = related ? 1 : 0.72;
      const total = entry.rxShown + entry.txShown;
      const width = entry.measured
        ? (1.1 + rateWeight(total) * 1.3) * scale
        : (0.9 + Math.min(entry.conns, 6) * 0.22) * scale;

      const path = () => {
        ctx.beginPath();
        ctx.moveTo(from.x, from.y);
        ctx.quadraticCurveTo(control.x, control.y, to.x, to.y);
      };

      // Soft glow under the edge.
      path();
      ctx.setLineDash([]);
      ctx.strokeStyle = rgba(style.color, 0.08 * entry.alpha * emphasis);
      ctx.lineWidth = width * 4;
      ctx.stroke();

      path();
      if (!entry.measured) {
        // Connections without byte counts: a marching dash shows activity
        // without claiming a direction or a rate.
        ctx.setLineDash([4 * scale, 7 * scale]);
        ctx.lineDashOffset = view.reducedMotion ? 0 : -seconds * 14;
      }
      ctx.strokeStyle = rgba(style.color, 0.55 * entry.alpha * emphasis);
      ctx.lineWidth = width;
      ctx.stroke();
      ctx.setLineDash([]);

      if (!entry.measured || view.reducedMotion) continue;
      // Particles: received bytes travel device -> this host, sent bytes
      // travel this host -> device. More and faster particles mean more bytes.
      for (const [rate, inbound] of [[entry.rxShown, true], [entry.txShown, false]]) {
        if (rate < 64) continue;
        const weight = rateWeight(rate);
        const count = clamp(Math.round(1 + weight * 1.6), 1, 8);
        const speed = 0.22 + weight * 0.12;
        const radius = (1.5 + weight * 0.55) * scale;
        for (let i = 0; i < count; i += 1) {
          const t = (seconds * speed + i / count + entry.phase + (inbound ? 0.5 / count : 0)) % 1;
          const point = bezierPoint(from, control, to, inbound ? 1 - t : t);
          const edgeFade = Math.min(1, t * 6, (1 - t) * 6);
          ctx.beginPath();
          ctx.arc(point.x, point.y, radius * 2.4, 0, TAU);
          ctx.fillStyle = rgba(style.color, 0.14 * entry.alpha * edgeFade);
          ctx.fill();
          ctx.beginPath();
          ctx.arc(point.x, point.y, radius, 0, TAU);
          ctx.fillStyle = rgba(inbound ? style.color : "#ffffff", 0.9 * entry.alpha * edgeFade);
          ctx.fill();
        }
      }
    }
    ctx.restore();
  }

  function renderPanelTraffic() {
    const target = panel.querySelector(".panel-traffic");
    if (!target || !view.selectedDeviceId) return;
    const node = view.nodes.get(view.selectedDeviceId);
    if (!node) return;
    const plural = (n) => `${n} connection${n === 1 ? "" : "s"}`;
    let rows = [];
    if (node.data.self) {
      const active = [...view.traffic.values()].filter((entry) => entry.active);
      if (active.length) {
        const peers = new Set(active.map((entry) => entry.deviceId)).size;
        rows.push(`<div><dt>Live traffic</dt><dd>${peers} device${peers === 1 ? "" : "s"} active</dd></div>`);
      }
    } else {
      rows = [...view.traffic.values()]
        .filter((entry) => entry.deviceId === node.id && entry.active)
        .sort((a, b) => a.via.localeCompare(b.via))
        .map((entry) => {
          const parts = [];
          if (entry.measured) parts.push(`↓ ${formatRate(entry.rx)} · ↑ ${formatRate(entry.tx)}`);
          if (entry.conns) parts.push(plural(entry.conns));
          return `<div><dt>${escapeHTML(TRAFFIC[entry.via].label)} traffic</dt><dd>${escapeHTML(parts.join(" · ") || "idle")}</dd></div>`;
        });
    }
    target.hidden = rows.length === 0;
    target.innerHTML = rows.join("");
  }

  function handleNodeClick(node) {
    if (!node) {
      closePanel();
      return;
    }
    selectNode(node);
  }

  // Reset drops every manual position and replays the automatic seed and
  // physics pass, then frames the result.
  function reseedAutomaticLayout() {
    for (const node of view.nodes.values()) {
      if (node.type !== "device") continue;
      const origin = automaticDeviceOrigin(node.data);
      node.data = { ...node.data, x: 0, y: 0, pinned: false };
      node.seed = origin.seed;
      moveDeviceTo(node, origin.x, origin.y);
    }
    for (let i = 0; i < 300; i += 1) physicsStep(1 / 60);
    fitView(true);
  }

  async function resetLayout() {
    if (!settingsResetBtn || settingsResetBtn.disabled) return;
    settingsResetBtn.disabled = true;
    try {
      const response = await fetch("/api/layout/reset", { method: "POST" });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      clearNodeDrag();
      reseedAutomaticLayout();
      closeSettings();
      showToast("Layout reset");
    } catch (_error) {
      showToast("Could not reset layout");
    } finally {
      settingsResetBtn.disabled = false;
    }
  }

  // Moving a device carries its service children, so the orbit ring stays
  // rigid until physics has a chance to settle it around the new centre.
  function moveDeviceTo(node, x, y) {
    const dx = x - node.x;
    const dy = y - node.y;
    if (!dx && !dy) return;
    node.x = x;
    node.y = y;
    for (const child of view.nodes.values()) {
      if (child.type === "service" && child.parentId === node.id) {
        child.x += dx;
        child.y += dy;
      }
    }
  }

  // A service bubble drags the device it belongs to, because the position
  // endpoint stores host positions, not individual service offsets.
  function beginNodeDrag(node, event) {
    if (!node) return false;
    const device = node.type === "service" ? view.nodes.get(node.parentId) : node;
    if (!device || device.type !== "device") return false;
    view.dragNodeId = device.id;
    view.nodeDragStart = {
      x: device.x,
      y: device.y,
      pointerX: event.clientX,
      pointerY: event.clientY,
    };
    return true;
  }

  function clearNodeDrag() {
    view.dragNodeId = null;
    view.nodeDragStart = null;
  }

  function cancelNodeDrag() {
    if (!view.dragNodeId) return;
    const node = view.nodes.get(view.dragNodeId);
    if (node && view.nodeDragStart) {
      moveDeviceTo(node, view.nodeDragStart.x, view.nodeDragStart.y);
    }
    clearNodeDrag();
  }

  function updateNodeDrag(event) {
    if (!view.dragNodeId || !view.nodeDragStart) return false;
    const start = view.nodeDragStart;
    const dx = event.clientX - start.pointerX;
    const dy = event.clientY - start.pointerY;
    if (!view.dragMoved && Math.hypot(dx, dy) <= DRAG_THRESHOLD) return true;
    view.dragMoved = true;
    const node = view.nodes.get(view.dragNodeId);
    if (!node) {
      clearNodeDrag();
      return false;
    }
    moveDeviceTo(node, start.x + dx / view.camera.zoom, start.y + dy / view.camera.zoom);
    view.target = null;
    view.userAdjusted = true;
    return true;
  }

  function saveDevicePosition(node) {
    const x = node.x;
    const y = node.y;
    node.data = { ...node.data, x, y, pinned: true };
    fetch(`/api/devices/${encodeURIComponent(node.id)}/position`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ x, y }),
    }).then((response) => {
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
    }).catch(() => {
      showToast("Could not save position");
    });
  }

  function endNodeDrag(event) {
    if (!view.dragNodeId) return;
    const node = view.nodes.get(view.dragNodeId);
    if (node && view.nodeDragStart) {
      if (event.type === "pointercancel") {
        moveDeviceTo(node, view.nodeDragStart.x, view.nodeDragStart.y);
      } else if (view.dragMoved) {
        saveDevicePosition(node);
      }
    }
    clearNodeDrag();
  }

  function endPointer(event) {
    view.pointers.delete(event.pointerId);
    if (view.pointers.size === 0) {
      const moved = view.dragMoved;
      const cancelled = event.type === "pointercancel";
      const clickTarget = moved || cancelled ? undefined : view.pointerDownHit;
      endNodeDrag(event);
      view.dragging = false;
      view.dragStart = null;
      canvas.classList.remove("dragging");
      view.pointerDownHit = null;
      view.pinchDistance = 0;
      view.lastPinchCenter = null;
      if (clickTarget !== undefined) {
        handleNodeClick(clickTarget);
        if (clickTarget) {
          hideTooltip();
          view.hoverId = clickTarget.id;
          canvas.style.cursor = "pointer";
        } else {
          updateHover(event.clientX, event.clientY);
        }
        return;
      }
      updateHover(event.clientX, event.clientY);
      return;
    }
    if (view.pointers.size === 1) {
      const remaining = [...view.pointers.values()][0];
      view.dragStart = {
        x: remaining.x,
        y: remaining.y,
        cameraX: view.camera.x,
        cameraY: view.camera.y,
      };
      view.dragging = true;
      view.dragMoved = true;
      view.pinchDistance = 0;
      view.lastPinchCenter = null;
    }
  }

  canvas.addEventListener("pointerdown", (event) => {
    if (event.pointerType === "mouse" && event.button !== 0) return;
    canvas.setPointerCapture(event.pointerId);
    view.pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
    if (view.pointers.size === 1) {
      view.dragging = true;
      view.dragMoved = false;
      view.pointerDownHit = hitTest(event.clientX, event.clientY);
      if (beginNodeDrag(view.pointerDownHit, event)) {
        view.dragStart = null;
      } else {
        view.dragStart = {
          x: event.clientX,
          y: event.clientY,
          cameraX: view.camera.x,
          cameraY: view.camera.y,
        };
      }
      canvas.classList.add("dragging");
      hideTooltip();
      return;
    }
    if (view.pointers.size === 2) {
      const points = [...view.pointers.values()];
      view.pinchDistance = Math.hypot(points[0].x - points[1].x, points[0].y - points[1].y);
      view.lastPinchCenter = {
        x: (points[0].x + points[1].x) / 2,
        y: (points[0].y + points[1].y) / 2,
      };
      cancelNodeDrag();
      view.dragMoved = true;
      view.dragging = false;
      canvas.classList.remove("dragging");
    }
  });

  canvas.addEventListener("pointermove", (event) => {
    if (view.pointers.has(event.pointerId)) {
      view.pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
    }
    view.pointer = { x: event.clientX, y: event.clientY };

    if (view.pointers.size === 2) {
      const points = [...view.pointers.values()];
      const distance = Math.hypot(points[0].x - points[1].x, points[0].y - points[1].y);
      const center = {
        x: (points[0].x + points[1].x) / 2,
        y: (points[0].y + points[1].y) / 2,
      };
      if (view.lastPinchCenter) {
        view.camera.x -= (center.x - view.lastPinchCenter.x) / view.camera.zoom;
        view.camera.y -= (center.y - view.lastPinchCenter.y) / view.camera.zoom;
      }
      if (view.pinchDistance > 0) {
        zoomAt(center.x, center.y, distance / view.pinchDistance);
      }
      view.pinchDistance = distance;
      view.lastPinchCenter = center;
      view.userAdjusted = true;
      return;
    }

    if (view.dragNodeId) {
      updateNodeDrag(event);
      return;
    }

    if (view.dragging && view.dragStart) {
      const dx = event.clientX - view.dragStart.x;
      const dy = event.clientY - view.dragStart.y;
      if (Math.hypot(dx, dy) > DRAG_THRESHOLD) view.dragMoved = true;
      view.camera.x = view.dragStart.cameraX - dx / view.camera.zoom;
      view.camera.y = view.dragStart.cameraY - dy / view.camera.zoom;
      view.target = null;
      view.userAdjusted = true;
      return;
    }

    updateHover(event.clientX, event.clientY);
  });

  canvas.addEventListener("pointerup", endPointer);
  canvas.addEventListener("pointercancel", endPointer);
  canvas.addEventListener("pointerleave", () => {
    if (view.pointers.size === 0) {
      view.hoverId = null;
      hideTooltip();
      canvas.style.cursor = "grab";
    }
  });

  canvas.addEventListener(
    "wheel",
    (event) => {
      event.preventDefault();
      const factor = Math.exp(-event.deltaY * 0.0013);
      zoomAt(event.clientX, event.clientY, factor);
      hideTooltip();
    },
    { passive: false },
  );

  canvas.addEventListener("dblclick", (event) => {
    const node = hitTest(event.clientX, event.clientY);
    if (!node) {
      fitView(true);
      return;
    }
    const target = node.type === "service" ? view.nodes.get(node.parentId) : node;
    if (!target) return;
    view.target = focusTarget(
      target.x,
      target.y,
      clamp(Math.max(view.camera.zoom, 1.65), 0.2, 3.2),
    );
    view.userAdjusted = true;
  });

  scanBtn.addEventListener("click", async () => {
    scanBtn.disabled = true;
    try {
      const response = await fetch("/api/scan", { method: "POST" });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      showToast("Scan requested");
    } catch (_error) {
      showToast("Could not request a scan");
      if (view.snapshot) syncChrome();
    }
  });

  historyBtn.addEventListener("click", () => {
    setHistoryVisible(!view.showOffline);
  });

  if (devicesBtn) {
    devicesBtn.addEventListener("click", toggleDeviceSheet);
  }

  if (deviceList) {
    deviceList.addEventListener("focusin", () => {
      if (view.deviceSheetOpen) return;
      view.deviceSheetOpen = true;
      syncDeviceSheet();
    });
    deviceList.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        closeDeviceSheet();
        return;
      }
      if (event.key === "Tab") {
        trapDeviceSheetFocus(event);
        return;
      }
      moveDeviceSheetFocus(event);
    });
    deviceList.addEventListener("click", (event) => {
      const entry = event.target.closest(".device-entry");
      if (!entry) return;
      const node = view.nodes.get(entry.dataset.deviceId);
      if (!node) return;
      closeDeviceSheet(false);
      selectNode(node, devicesBtn);
      const close = panel.querySelector(".panel-close");
      if (close) close.focus();
    });
  }

  if (deviceListClose) {
    deviceListClose.addEventListener("click", () => closeDeviceSheet());
  }

  if (tailnetBtn) {
    tailnetBtn.addEventListener("click", () => {
      setTailnetVisible(!view.showTailnet);
    });
  }

  settingsBtn.addEventListener("click", openSettings);
  settingsClose.addEventListener("click", closeSettings);
  settingsHistoryBtn.addEventListener("click", () => {
    setHistoryVisible(!view.showOffline, false);
  });
  settingsTailnetBtn.addEventListener("click", () => {
    setTailnetVisible(!view.showTailnet, false);
  });
  settingsResetBtn.addEventListener("click", resetLayout);
  settingsDialog.addEventListener("cancel", (event) => {
    event.preventDefault();
    closeSettings();
  });
  settingsDialog.addEventListener("click", (event) => {
    if (event.target === settingsDialog) closeSettings();
  });
  settingsDialog.addEventListener("keydown", trapSettingsFocus);

  document.getElementById("zoomIn").addEventListener("click", () => {
    zoomAt((size.width - size.insetRight) / 2, size.height / 2, 1.25);
  });
  document.getElementById("zoomOut").addEventListener("click", () => {
    zoomAt((size.width - size.insetRight) / 2, size.height / 2, 1 / 1.25);
  });
  document.getElementById("fitBtn").addEventListener("click", () => {
    fitView(true);
  });

  panel.addEventListener("click", (event) => {
    if (event.target.closest(".panel-dock")) {
      setDockDetails(!view.dockDetails);
      return;
    }
    if (event.target.closest(".panel-close")) {
      closePanel();
      return;
    }
    const row = event.target.closest(".service-row");
    if (!row) return;
    const serviceId = row.dataset.serviceId;
    view.selectedServiceId = serviceId;
    const node = view.nodes.get(serviceId);
    if (node) {
      view.target = focusTarget(node.x, node.y, Math.max(view.camera.zoom, 1.45));
      view.userAdjusted = true;
    }
    renderPanel();
  });

  window.addEventListener("keydown", (event) => {
    if (event.target.matches("input, textarea, select, [contenteditable]")) return;
    if (deviceList && deviceList.contains(event.target)) return;
    if (settingsDialog.open) return;
    if (event.key === "Escape") {
      if (view.deviceSheetOpen) {
        closeDeviceSheet();
        return;
      }
      closePanel();
      hideTooltip();
      return;
    }
    if (event.key === "+" || event.key === "=") {
      zoomAt((size.width - size.insetRight) / 2, size.height / 2, 1.25);
      return;
    }
    if (event.key === "-" || event.key === "_") {
      zoomAt((size.width - size.insetRight) / 2, size.height / 2, 1 / 1.25);
      return;
    }
    if (event.key === "0" || event.key.toLowerCase() === "f") {
      fitView(true);
      return;
    }
    if (event.key.toLowerCase() === "l" && !event.metaKey && !event.ctrlKey && !event.altKey) {
      toggleDeviceSheet();
      return;
    }
    if (event.key.toLowerCase() === "s" && !event.metaKey && !event.ctrlKey && !event.altKey) {
      openSettings();
    }
  });

  function setConnection(state) {
    conn.classList.toggle("off", state === "off");
    conn.classList.toggle("connecting", state === "connecting");
    const label = conn.querySelector("span");
    if (state === "live") label.textContent = "live";
    else if (state === "off") label.textContent = "reconnecting";
    else label.textContent = "connecting";
  }

  let eventSource = null;
  function connectEvents() {
    if (eventSource) eventSource.close();
    eventSource = new EventSource("/api/events");
    eventSource.onopen = () => setConnection("live");
    eventSource.onerror = () => setConnection("off");
    eventSource.addEventListener("state", (event) => {
      try {
        applySnapshot(JSON.parse(event.data));
        setConnection("live");
      } catch (error) {
        console.error("bad snapshot", error);
      }
    });
    eventSource.addEventListener("traffic", (event) => {
      try {
        applyTraffic(JSON.parse(event.data));
      } catch (error) {
        console.error("bad traffic sample", error);
      }
    });
  }

  async function loadState() {
    try {
      const response = await fetch("/api/state", { cache: "no-store" });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      applySnapshot(await response.json());
      setConnection("live");
    } catch (_error) {
      setConnection("off");
    }
  }

  let lastFrame = performance.now();
  function frame(now) {
    const dt = clamp((now - lastFrame) / 1000, 0.001, 0.05);
    lastFrame = now;
    updateCamera(dt);
    // Layout settles when a snapshot arrives; stepping physics every frame
    // made the bubbles jitter at low frame rates.
    for (const node of view.nodes.values()) {
      const target = view.hoverId === node.id ? 1 : 0;
      node.hover += (target - node.hover) * Math.min(1, dt * 12);
    }
    updateTraffic(dt);
    draw(now);
    window.requestAnimationFrame(frame);
  }

  window.addEventListener("resize", resize);
  window.setInterval(updateSubtitle, 1000);
  window.setInterval(refreshServiceTimes, 15000);
  window.setInterval(() => {
    if (!eventSource || eventSource.readyState !== 1) loadState();
  }, 15000);

  resize();
  renderLegend();
  setConnection("connecting");
  loadState();
  connectEvents();
  window.requestAnimationFrame(frame);
})();
