(() => {
  "use strict";

  const TAU = Math.PI * 2;
  const SERVICE_R = 8.5;
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
  const conn = document.getElementById("conn");
  const legend = document.getElementById("legend");
  const empty = document.getElementById("empty");

  const size = {
    width: window.innerWidth,
    height: window.innerHeight,
    dpr: Math.min(window.devicePixelRatio || 1, 2),
    chromeTop: 94,
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
    tailnetPresent: false,
    territories: null,
    territoryVersion: 0,
    territorySig: "",
    territoryLayer: null,
    dragging: false,
    dragMoved: false,
    dragStart: null,
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
  };

  try {
    if (window.localStorage.getItem("hearthlight.history") === "0") {
      view.showOffline = false;
    }
    if (window.localStorage.getItem("hearthlight.tailnet") === "0") {
      view.showTailnet = false;
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

  function resize() {
    size.width = window.innerWidth;
    size.height = window.innerHeight;
    size.dpr = Math.min(window.devicePixelRatio || 1, 2);
    const topbar = document.querySelector(".topbar");
    size.chromeTop = topbar ? topbar.getBoundingClientRect().bottom + 10 : 94;
    canvas.width = Math.round(size.width * size.dpr);
    canvas.height = Math.round(size.height * size.dpr);
    if (view.snapshot && !view.userAdjusted) {
      fitView(false);
    }
  }

  function worldToScreen(x, y) {
    return {
      x: (x - view.camera.x) * view.camera.zoom + size.width / 2,
      y: (y - view.camera.y) * view.camera.zoom + size.height / 2,
    };
  }

  function screenToWorld(x, y) {
    return {
      x: (x - size.width / 2) / view.camera.zoom + view.camera.x,
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
    const narrow = size.width < 700;
    const horizontalPad = narrow ? 26 : 90;
    const verticalPad = narrow ? Math.max(260, size.chromeTop + 150) : 230;
    const zoom = clamp(
      Math.min((size.width - horizontalPad) / width, (size.height - verticalPad) / height),
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

  function convexHull(points) {
    if (points.length <= 3) return points.slice();
    const sorted = points.slice().sort((a, b) => a.x - b.x || a.y - b.y);
    const cross = (o, a, b) => (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x);
    const lower = [];
    for (const point of sorted) {
      while (lower.length >= 2 && cross(lower[lower.length - 2], lower[lower.length - 1], point) <= 0) lower.pop();
      lower.push(point);
    }
    const upper = [];
    for (let i = sorted.length - 1; i >= 0; i -= 1) {
      const point = sorted[i];
      while (upper.length >= 2 && cross(upper[upper.length - 2], upper[upper.length - 1], point) <= 0) upper.pop();
      upper.push(point);
    }
    lower.pop();
    upper.pop();
    return lower.concat(upper);
  }

  // territoryHull pads every member host by its bubble plus service ring, then
  // takes the convex hull so the boundary always tracks pan and zoom.
  function territoryHull(nodes, pad) {
    const points = [];
    for (const node of nodes) {
      const radius = node.baseR + pad;
      for (let i = 0; i < 14; i += 1) {
        const angle = (i / 14) * TAU;
        points.push({ x: node.x + Math.cos(angle) * radius, y: node.y + Math.sin(angle) * radius });
      }
    }
    return convexHull(points);
  }

  function updateTerritories() {
    const devices = [...view.nodes.values()].filter((node) => node.type === "device");
    const local = devices.filter((node) => hasLocal(node.data));
    const tailnet = devices.filter((node) => hasTailnet(node.data));
    view.territories = {
      lan: local.length ? territoryHull(local, 56) : null,
      tailnet: tailnet.length ? territoryHull(tailnet, 56) : null,
      lanCount: local.length,
      tailnetCount: tailnet.length,
    };
    view.territoryVersion += 1;
    view.territorySig = "";
  }

  function hullScreen(points) {
    return points.map((point) => worldToScreen(point.x, point.y));
  }

  // tracedPath rounds a convex hull by joining edge midpoints with quadratic
  // curves through the vertices. The result is a smooth closed membrane.
  function tracedPath(target, points) {
    const n = points.length;
    if (n < 3) return;
    const mid = (a, b) => ({ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 });
    const start = mid(points[n - 1], points[0]);
    target.beginPath();
    target.moveTo(start.x, start.y);
    for (let i = 0; i < n; i += 1) {
      const current = points[i];
      const next = points[(i + 1) % n];
      const end = mid(current, next);
      target.quadraticCurveTo(current.x, current.y, end.x, end.y);
    }
    target.closePath();
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

  function drawTerritoryFill(target, points, tailnet) {
    const bounds = pointsBounds(points);
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
    tracedPath(target, points);
    target.fill();
    target.restore();

    target.save();
    target.fillStyle = tailnet ? "rgba(94, 231, 255, 0.028)" : "rgba(255, 209, 102, 0.022)";
    tracedPath(target, points);
    target.fill();
    target.restore();
  }

  function drawAuroraRibbon(target, lan, tailnet) {
    const from = pointsBounds(lan);
    const to = pointsBounds(tailnet);
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
    tracedPath(target, lan);
    target.clip();
    tracedPath(target, tailnet);
    target.clip();
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
      size.width, size.height, view.camera.x.toFixed(2), view.camera.y.toFixed(2),
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
      const lan = territories.lan ? hullScreen(territories.lan) : null;
      const tailnet = territories.tailnet ? hullScreen(territories.tailnet) : null;
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

  function drawBoundary(points, color, glow) {
    ctx.save();
    ctx.setLineDash([1.5, 7]);
    ctx.lineWidth = 1.2;
    ctx.strokeStyle = color;
    ctx.shadowColor = glow;
    ctx.shadowBlur = 9;
    tracedPath(ctx, points);
    ctx.stroke();
    ctx.restore();
  }

  function drawTerritoryBoundaries() {
    if (!view.territories) return;
    if (view.territories.lan) {
      drawBoundary(hullScreen(view.territories.lan), "rgba(255, 226, 166, 0.66)", "rgba(255, 209, 102, 0.42)");
    }
    if (view.territories.tailnet) {
      drawBoundary(hullScreen(view.territories.tailnet), "rgba(126, 240, 255, 0.82)", "rgba(94, 231, 255, 0.62)");
    }
  }

  function topPoint(points) {
    let best = null;
    for (const point of points) {
      const screen = worldToScreen(point.x, point.y);
      if (!best || screen.y < best.y) best = screen;
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
    const name = truncate(device.hostname || device.ip, 24);
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
    const info = view.snapshot.tailnet;
    const suffix = info && (info.magicDnsSuffix || info.name) ? (info.magicDnsSuffix || info.name) : "tailnet";
    const peers = info && info.peersTotal ? info.peersTotal : tailnetCount;
    const placed = [];
    if (lan) {
      const top = topPoint(lan);
      placed.push(drawRegionPill(top.x, top.y, "HOME LAN", `${network} · ${lanCount} HOSTS`, false, placed));
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
          displacement.get(a.id).x -= px;
          displacement.get(a.id).y -= py;
          displacement.get(b.id).x += px;
          displacement.get(b.id).y += py;
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
      const move = displacement.get(device.id);
      if (view.tailnetPresent) {
        // Bias the two memberships into left and right territories, with
        // dual-membership hosts held near the center so the lens sits between.
        const reach = deviceReach(device.data);
        const lane = reach === "local" ? -600 : reach === "tailnet" ? 620 : 0;
        const spread = reach === "both" ? 140 : 210;
        const targetX = lane + (device.seed - 0.5) * spread * 2;
        move.x += (targetX - device.x) * 0.012;
      }
      move.x -= device.x * 0.0012;
      move.y -= device.y * 0.0012;
    }

    const step = clamp(dt * 60, 0, 2);
    for (const node of nodes) {
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
      if (!node) {
        const seed = hashString(device.id);
        const angle = ((seed % 3600) / 3600) * TAU;
        const radius = 120 + (seed % 240);
        const reach = deviceReach(device);
        const bias = !view.tailnetPresent ? 0 : reach === "local" ? -600 : reach === "tailnet" ? 620 : 0;
        node = {
          id: device.id,
          type: "device",
          data: device,
          x: Math.cos(angle) * radius * 0.45 + bias,
          y: Math.sin(angle) * radius,
          seed: (seed % 1000) / 1000,
          baseR: 30,
          r: 30,
          hover: 0,
        };
        view.nodes.set(device.id, node);
      }
      node.data = device;
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
        index += 1;
      }
    }

    for (const [id, node] of view.nodes) {
      if (node.type === "device" && !presentDevices.has(id)) {
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
    const tailnet = snapshot.tailnet && (snapshot.tailnet.magicDnsSuffix || snapshot.tailnet.name)
      ? (snapshot.tailnet.magicDnsSuffix || snapshot.tailnet.name)
      : "";
    const scanned = validTime(snapshot.lastScan) ? `last scan ${relativeTime(snapshot.lastScan)}` : "waiting for first scan";
    const prefix = snapshot.scanning ? "scanning · " : "";
    const error = snapshot.lastError ? " · scan error" : "";
    subtitle.textContent = `${prefix}${[networks, tailnet, scanned].filter(Boolean).join(" · ")}${error}`;
    subtitle.title = snapshot.lastError || [networks, tailnet].filter(Boolean).join(" · ");
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
    `;
  }

  function openPanel(deviceId) {
    view.selectedDeviceId = deviceId;
    renderPanel();
    panel.classList.add("open");
    panel.setAttribute("aria-hidden", "false");
  }

  function closePanel() {
    view.selectedDeviceId = null;
    view.selectedServiceId = null;
    panel.classList.remove("open");
    panel.setAttribute("aria-hidden", "true");
  }

  function renderPanel() {
    const snapshot = view.snapshot;
    if (!snapshot || !view.selectedDeviceId) return;
    const device = snapshot.devices.find((candidate) => candidate.id === view.selectedDeviceId);
    if (!device) return;
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
        return `
          <li class="service-row ${service.online ? "on" : "off"} ${selected ? "selected" : ""}" data-service-id="${escapeHTML(service.id)}">
            <span class="svc-dot" style="--c:${color}"></span>
            <span class="svc-main"><strong>${escapeHTML(name)}</strong><span>${escapeHTML(sub)}</span></span>
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
      <button class="panel-close" type="button" aria-label="Close details">×</button>
      <div class="panel-kicker">
        <span class="status-dot ${device.online ? "on" : "off"}"></span>
        ${device.online ? "online" : "history"} · ${escapeHTML(kind.label)}${view.tailnetPresent ? ` · ${escapeHTML(reachLabel(reach))}` : ""}
      </div>
      <h2>${escapeHTML(device.hostname || device.ip)}</h2>
      <p class="ip">${escapeHTML(device.ip)}${device.self ? " · this host" : ""}</p>
      <dl class="meta">
        ${meta.join("")}
      </dl>
      <h3>Services <span>${onlineCount}/${services.length} up</span></h3>
      ${rows || '<div class="empty-services">No TCP services found on this device.</div>'}
    `;
  }

  function selectNode(node) {
    if (!node) return;
    hideTooltip();
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
      const services = device.services || [];
      const online = services.filter((service) => service.online).length;
      const kind = KIND[device.kind] || KIND.device;
      const reach = deviceReach(device);
      const tailnet = device.tailnetIp
        ? `<div class="tip-meta">${escapeHTML(device.tailnetIp)}${device.tailnetOs ? ` · ${escapeHTML(device.tailnetOs)}` : ""}${device.tailnetName ? ` · ${escapeHTML(device.tailnetName)}` : ""}</div>`
        : "";
      tooltip.innerHTML = `
        <strong>${escapeHTML(device.hostname || device.ip)}</strong>
        <div>${escapeHTML(device.ip)} · ${escapeHTML(kind.label)}${device.online ? "" : " · history"}</div>
        <div class="tip-meta">${view.tailnetPresent ? `${escapeHTML(reachLabel(reach))} · ` : ""}${online}/${services.length} services up${device.vendor ? ` · ${escapeHTML(device.vendor)}` : ""}</div>
        ${tailnet}
      `;
    } else {
      const parent = view.nodes.get(node.parentId);
      const serviceReach = node.data.tailnet && node.data.local ? "both" : node.data.tailnet ? "tailnet" : "local";
      tooltip.innerHTML = `
        <strong>${escapeHTML(node.data.product || node.data.name || "service")}</strong>
        <div>${escapeHTML(node.data.name || "TCP")} · port ${node.data.port}</div>
        <div class="tip-meta">${parent ? `${escapeHTML(parent.data.hostname || parent.data.ip)} · ` : ""}${node.data.online ? "online" : "history"}${view.tailnetPresent ? ` · ${escapeHTML(reachLabel(serviceReach))}` : ""}</div>
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

  function roundRectPath(x, y, width, height, radius) {
    const r = Math.min(radius, Math.abs(width) / 2, Math.abs(height) / 2);
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.arcTo(x + width, y, x + width, y + height, r);
    ctx.arcTo(x + width, y + height, x, y + height, r);
    ctx.arcTo(x, y + height, x, y, r);
    ctx.arcTo(x, y, x + width, y, r);
    ctx.closePath();
  }

  function drawDeviceIcon(kind, x, y, iconSize, color) {
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
          roundRectPath(-u * 0.75, row * u * 0.52 - u * 0.19, u * 1.5, u * 0.36, u * 0.08);
          ctx.stroke();
        }
        ctx.beginPath();
        ctx.arc(u * 0.45, -u * 0.01, u * 0.07, 0, TAU);
        ctx.fill();
        break;
      case "computer":
        roundRectPath(-u * 0.9, -u * 0.72, u * 1.8, u * 1.16, u * 0.12);
        ctx.stroke();
        ctx.beginPath();
        ctx.moveTo(-u * 0.38, u * 0.62);
        ctx.lineTo(u * 0.38, u * 0.62);
        ctx.moveTo(0, u * 0.44);
        ctx.lineTo(0, u * 0.62);
        ctx.stroke();
        break;
      case "phone":
        roundRectPath(-u * 0.36, -u * 0.8, u * 0.72, u * 1.6, u * 0.16);
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
        roundRectPath(-u * 0.44, -u * 0.86, u * 0.88, u * 0.58, u * 0.06);
        ctx.stroke();
        roundRectPath(-u * 0.85, -u * 0.26, u * 1.7, u * 0.68, u * 0.12);
        ctx.stroke();
        roundRectPath(-u * 0.44, u * 0.16, u * 0.88, u * 0.58, u * 0.05);
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
        roundRectPath(-u * 0.5, -u * 0.5, u, u, u * 0.15);
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

  function strokeLabel(text, x, y, fontSize) {
    ctx.lineWidth = Math.max(3, fontSize * 0.28);
    ctx.strokeStyle = "rgba(3, 7, 12, 0.86)";
    ctx.lineJoin = "round";
    ctx.strokeText(text, x, y);
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

    ctx.save();
    ctx.globalAlpha = historicAlpha;
    drawDeviceIcon(device.kind, point.x, point.y, radius * 1.52, rgba(color, active ? 0.92 : 0.55));
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

    const name = truncate(device.hostname || device.ip, 24);
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

      ctx.beginPath();
      ctx.arc(point.x, point.y, Math.max(1.2, radius * 0.28), 0, TAU);
      ctx.fillStyle = rgba(color, 0.95);
      ctx.fill();
    } else {
      ctx.setLineDash([2.5, 3]);
      ctx.beginPath();
      ctx.arc(point.x, point.y, radius, 0, TAU);
      ctx.strokeStyle = rgba(color, 0.55);
      ctx.lineWidth = 1;
      ctx.stroke();
      ctx.setLineDash([]);
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

  function handleNodeClick(node) {
    if (!node) {
      closePanel();
      return;
    }
    selectNode(node);
  }

  function endPointer(event) {
    view.pointers.delete(event.pointerId);
    if (view.pointers.size === 0) {
      const moved = view.dragMoved;
      const clickTarget = moved ? undefined : view.pointerDownHit;
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
      view.dragStart = {
        x: event.clientX,
        y: event.clientY,
        cameraX: view.camera.x,
        cameraY: view.camera.y,
      };
      view.pointerDownHit = hitTest(event.clientX, event.clientY);
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

    if (view.dragging && view.dragStart) {
      const dx = event.clientX - view.dragStart.x;
      const dy = event.clientY - view.dragStart.y;
      if (Math.hypot(dx, dy) > 4) view.dragMoved = true;
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
    view.target = {
      x: target.x,
      y: target.y,
      zoom: clamp(Math.max(view.camera.zoom, 1.65), 0.2, 3.2),
    };
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
    view.showOffline = !view.showOffline;
    historyBtn.setAttribute("aria-pressed", String(view.showOffline));
    try {
      window.localStorage.setItem("hearthlight.history", view.showOffline ? "1" : "0");
    } catch (_error) {
      // Storage is optional.
    }
    showToast(view.showOffline ? "History shown" : "History hidden");
    hideTooltip();
  });

  if (tailnetBtn) {
    tailnetBtn.addEventListener("click", () => {
      view.showTailnet = !view.showTailnet;
      tailnetBtn.setAttribute("aria-pressed", String(view.showTailnet));
      try {
        window.localStorage.setItem("hearthlight.tailnet", view.showTailnet ? "1" : "0");
      } catch (_error) {
        // Storage is optional.
      }
      view.territorySig = "";
      showToast(view.showTailnet ? "Tailnet shown" : "Tailnet hidden");
      hideTooltip();
      syncChrome();
    });
  }

  document.getElementById("zoomIn").addEventListener("click", () => {
    zoomAt(size.width / 2, size.height / 2, 1.25);
  });
  document.getElementById("zoomOut").addEventListener("click", () => {
    zoomAt(size.width / 2, size.height / 2, 1 / 1.25);
  });
  document.getElementById("fitBtn").addEventListener("click", () => {
    fitView(true);
  });

  panel.addEventListener("click", (event) => {
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
      view.target = {
        x: node.x,
        y: node.y,
        zoom: Math.max(view.camera.zoom, 1.45),
      };
      view.userAdjusted = true;
    }
    renderPanel();
  });

  window.addEventListener("keydown", (event) => {
    if (event.target.matches("input, textarea, select")) return;
    if (event.key === "Escape") {
      closePanel();
      hideTooltip();
      return;
    }
    if (event.key === "+" || event.key === "=") {
      zoomAt(size.width / 2, size.height / 2, 1.25);
      return;
    }
    if (event.key === "-" || event.key === "_") {
      zoomAt(size.width / 2, size.height / 2, 1 / 1.25);
      return;
    }
    if (event.key === "0" || event.key.toLowerCase() === "f") {
      fitView(true);
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
    draw(now);
    window.requestAnimationFrame(frame);
  }

  window.addEventListener("resize", resize);
  window.setInterval(updateSubtitle, 1000);
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
