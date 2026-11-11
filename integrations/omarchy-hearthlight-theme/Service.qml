import QtQuick
import Quickshell
import Quickshell.Io

Item {
  id: root

  readonly property string home: Quickshell.env("HOME")
  readonly property string statePath: home + "/.local/state/omarchy/current"
  readonly property string themeNamePath: statePath + "/theme.name"
  readonly property string colorsPath: statePath + "/theme/colors.toml"
  readonly property string generationPath: statePath + "/theme.generation"
  readonly property string configPath: home + "/.config/omarchy/hearthlight-theme.json"
  readonly property string defaultOutputPath: home + "/.config/hearthlight/omarchy-theme.json"

  readonly property var paletteKeys: [
    "background", "dark_background", "darker_background", "lighter_background",
    "foreground", "dark_foreground", "light_foreground", "bright_foreground",
    "accent", "selection", "muted",
    "red", "yellow", "orange", "green", "cyan", "blue", "magenta", "brown",
    "bright_red", "bright_yellow", "bright_orange", "bright_green",
    "bright_cyan", "bright_blue", "bright_magenta"
  ]

  // Stock Omarchy names and their Hearthlight variant ids. Unknown or custom
  // themes omit themeId so the server can synthesize a palette by name.
  readonly property var stockThemeIds: ({
    "catppuccin": "catppuccin-mocha",
    "catppuccin-latte": "catppuccin-latte",
    "ethereal": "ethereal",
    "everforest": "everforest-dark",
    "flexoki-light": "flexoki-light",
    "gruvbox": "gruvbox-dark",
    "hackerman": "hackerman",
    "jade-horse": "jade-horse",
    "kanagawa": "kanagawa-wave",
    "last-horizon": "last-horizon",
    "lumon": "lumon",
    "lupine": "lupine",
    "matte-black": "matte-black",
    "miasma": "miasma",
    "nord": "nord-dark",
    "retro-82": "retro-82",
    "ristretto": "ristretto",
    "rose-pine": "rose-pine-dawn",
    "solitude": "solitude",
    "starship": "starship",
    "tokyo-night": "tokyo-night",
    "vantablack": "vantablack",
    "white": "white"
  })

  property string outputPath: defaultOutputPath
  property string activeOutputPath: ""
  property string activeText: ""
  property string pendingText: ""
  property string tempPath: ""
  property int writeSerial: 0
  property bool writing: false
  property bool warnedConfig: false
  property bool warnedTheme: false
  property bool warnedWrite: false
  property bool warnedNetwork: false

  function warnConfigOnce(message) {
    if (root.warnedConfig) return
    root.warnedConfig = true
    console.warn("hearthlight-theme", message)
  }

  function warnThemeOnce(message) {
    if (root.warnedTheme) return
    root.warnedTheme = true
    console.warn("hearthlight-theme", message)
  }

  function warnWriteOnce(message) {
    if (root.warnedWrite) return
    root.warnedWrite = true
    console.warn("hearthlight-theme", message)
  }

  function warnNetworkOnce() {
    if (root.warnedNetwork) return
    root.warnedNetwork = true
    console.warn("hearthlight-theme", "hearthlightUrl is not used; this plugin only writes the local theme file")
  }

  function scheduleRefresh() {
    if (refreshTimer.running) refreshTimer.restart()
    else refreshTimer.start()
  }

  function scheduleReload() {
    if (reloadTimer.running) reloadTimer.restart()
    else reloadTimer.start()
  }

  function reloadThemeFiles() {
    themeNameFile.reload()
    colorsFile.reload()
    generationFile.reload()
  }

  function expandPath(value) {
    if (typeof value !== "string") return ""
    var path = value.trim()
    if (path.length === 0 || /[\u0000\n\r]/.test(path)) return ""
    if (path === "~") path = root.home
    else if (path.indexOf("~/") === 0) path = root.home + path.slice(1)
    if (path.length === 0 || path.charAt(0) !== "/" || path.charAt(path.length - 1) === "/") return ""
    return path
  }

  function applyConfig(raw) {
    var next = root.defaultOutputPath
    var text = String(raw || "").trim()
    var valid = true

    if (text.length > 0) {
      try {
        var parsed = JSON.parse(text)
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
          valid = false
          root.warnConfigOnce("Ignoring " + root.configPath + " because it is not a JSON object")
        } else {
          if (parsed.hearthlightUrl !== undefined && parsed.hearthlightUrl !== null
              && String(parsed.hearthlightUrl).length > 0)
            root.warnNetworkOnce()
          if (parsed.outputPath !== undefined && parsed.outputPath !== null
              && String(parsed.outputPath).length > 0) {
            var configured = root.expandPath(parsed.outputPath)
            if (configured.length > 0) next = configured
            else {
              valid = false
              root.warnConfigOnce("outputPath must be absolute or start with ~/, using " + root.defaultOutputPath)
            }
          }
        }
      } catch (error) {
        valid = false
        root.warnConfigOnce("Ignoring malformed JSON in " + root.configPath)
      }
    }

    if (valid) root.warnedConfig = false
    if (root.outputPath !== next) {
      root.outputPath = next
      root.scheduleRefresh()
    }
  }

  function parseTomlValue(raw) {
    var text = String(raw || "").trim()
    if (text.length === 0) return null
    var quote = text.charAt(0)
    if (quote === '"' || quote === "'") {
      var value = ""
      for (var index = 1; index < text.length; index++) {
        var char = text.charAt(index)
        if (char === "\\" && quote === '"' && index + 1 < text.length) {
          value += text.charAt(index + 1)
          index += 1
          continue
        }
        if (char === quote) return value
        value += char
      }
      return null
    }
    var comment = text.indexOf("#")
    if (comment >= 0) text = text.slice(0, comment)
    text = text.trim()
    return text.length > 0 ? text : null
  }

  function parseToml(raw) {
    var result = {}
    var lines = String(raw || "").split(/\r?\n/)
    for (var index = 0; index < lines.length; index++) {
      var line = lines[index].trim()
      if (line.length === 0 || line.charAt(0) === "#" || line.charAt(0) === "[") continue
      var equals = line.indexOf("=")
      if (equals <= 0) continue
      var key = line.slice(0, equals).trim()
      if (!/^[A-Za-z0-9_-]+$/.test(key)) continue
      var value = root.parseTomlValue(line.slice(equals + 1).trim())
      if (value !== null) result[key] = value
    }
    return result
  }

  function normalizeColor(value) {
    if (typeof value !== "string") return ""
    var color = value.trim().toLowerCase()
    if (!/^#[0-9a-f]+$/.test(color)) return ""
    var hex = color.slice(1)
    if (hex.length === 3 || hex.length === 4) {
      var expanded = ""
      for (var index = 0; index < hex.length; index++)
        expanded += hex.charAt(index) + hex.charAt(index)
      hex = expanded
    }
    if (hex.length !== 6 && hex.length !== 8) return ""
    return "#" + hex
  }

  function isDark(color) {
    if (typeof color !== "string" || color.length < 7) return true
    function linear(channel) {
      return channel <= 0.04045 ? channel / 12.92 : Math.pow((channel + 0.055) / 1.055, 2.4)
    }
    var red = linear(parseInt(color.slice(1, 3), 16) / 255)
    var green = linear(parseInt(color.slice(3, 5), 16) / 255)
    var blue = linear(parseInt(color.slice(5, 7), 16) / 255)
    return 0.2126 * red + 0.7152 * green + 0.0722 * blue < 0.5
  }

  function normalizePalette(raw) {
    var parsed = root.parseToml(raw)
    var resolved = {}
    for (var index = 0; index < root.paletteKeys.length; index++) {
      var key = root.paletteKeys[index]
      var color = root.normalizeColor(parsed[key])
      if (color.length > 0) resolved[key] = color
    }

    if (!resolved.background) {
      resolved.background = root.normalizeColor(parsed.dark_background)
        || root.normalizeColor(parsed.darker_background)
        || root.normalizeColor(parsed.lighter_background)
    }
    if (!resolved.foreground) {
      resolved.foreground = root.normalizeColor(parsed.bright_foreground)
        || root.normalizeColor(parsed.light_foreground)
        || root.normalizeColor(parsed.dark_foreground)
    }
    if (!resolved.accent) {
      resolved.accent = root.normalizeColor(parsed.blue)
        || root.normalizeColor(parsed.cyan)
        || resolved.foreground
    }
    if (!resolved.muted) {
      resolved.muted = root.normalizeColor(parsed.dark_foreground)
        || root.normalizeColor(parsed.selection)
        || resolved.foreground
    }
    if (!resolved.selection) {
      resolved.selection = root.normalizeColor(parsed.lighter_background)
        || resolved.accent
    }

    var colors = {}
    for (var index = 0; index < root.paletteKeys.length; index++) {
      var key = root.paletteKeys[index]
      if (resolved[key]) colors[key] = resolved[key]
    }

    var mode = String(parsed.mode || "").toLowerCase()
    if (mode !== "dark" && mode !== "light")
      mode = root.isDark(colors.background) ? "dark" : "light"

    return { mode: mode, colors: colors }
  }

  function displayName(name) {
    var parts = name.split(/[-_]+/)
    var words = []
    for (var index = 0; index < parts.length; index++) {
      var part = parts[index]
      if (part.length === 0) continue
      words.push(part.charAt(0).toUpperCase() + part.slice(1))
    }
    return words.length > 0 ? words.join(" ") : name
  }

  function parseGeneration(raw) {
    var text = String(raw || "").trim()
    if (!/^\d+$/.test(text)) return 0
    var value = Number(text)
    return isFinite(value) ? value : 0
  }

  function refresh() {
    if (root.home.length === 0) {
      root.warnThemeOnce("HOME is not set; cannot locate the Omarchy theme files")
      return
    }

    var name = String(themeNameFile.text() || "").split(/\r?\n/)[0].trim()
    var colorsRaw = String(colorsFile.text() || "")
    if (name.length === 0 || colorsRaw.length === 0) {
      root.warnThemeOnce("Waiting for theme files under " + root.statePath + "; keeping the last good file")
      return
    }

    var palette = root.normalizePalette(colorsRaw)
    if (!palette.colors.background || !palette.colors.foreground) {
      root.warnThemeOnce("No usable palette in " + root.statePath + "/theme/colors.toml; keeping the last good file")
      return
    }
    root.warnedTheme = false

    var payload = {
      schemaVersion: 1,
      name: name,
      displayName: root.displayName(name),
      mode: palette.mode,
      generation: root.parseGeneration(generationFile.text())
    }
    var themeId = root.stockThemeIds[name]
    if (themeId) payload.themeId = themeId
    payload.colors = palette.colors

    root.queueWrite(JSON.stringify(payload, null, 2) + "\n")
  }

  function dirname(path) {
    var slash = path.lastIndexOf("/")
    return slash > 0 ? path.slice(0, slash) : "/"
  }

  function queueWrite(text) {
    if (root.writing) {
      root.pendingText = text
      return
    }
    root.startWrite(text)
  }

  function startWrite(text) {
    if (root.home.length === 0 || root.outputPath.charAt(0) !== "/") {
      root.warnWriteOnce("Cannot write the Omarchy theme file without a valid output path")
      return
    }
    root.writing = true
    root.activeOutputPath = root.outputPath
    root.activeText = text
    root.pendingText = ""
    root.writeSerial += 1
    root.tempPath = root.activeOutputPath + ".tmp." + root.writeSerial
    // The palette travels as an argv value to a fixed script. It is never
    // interpolated into shell source.
    writeProcess.command = [
      "sh", "-c",
      'umask 077; mkdir -p -- "$1" || exit 1; printf "%s" "$4" > "$2" || exit 1; chmod 600 -- "$2" || exit 1; mv -f -- "$2" "$3"',
      "hearthlight-theme",
      root.dirname(root.activeOutputPath),
      root.tempPath,
      root.activeOutputPath,
      root.activeText
    ]
    writeProcess.running = true
  }

  function finishWrite() {
    root.writing = false
    root.warnedWrite = false
    var next = root.pendingText
    root.pendingText = ""
    if (next.length > 0) root.startWrite(next)
  }

  function failWrite(message) {
    root.writing = false
    root.pendingText = ""
    root.warnWriteOnce(message + "; keeping " + root.activeOutputPath)
    Quickshell.execDetached(["rm", "-f", "--", root.tempPath])
  }

  Timer {
    id: refreshTimer
    interval: 150
    repeat: false
    onTriggered: root.refresh()
  }

  // A theme switch replaces the current/theme directory. Reloading after the
  // directory settles re-arms the file watchers and avoids reading a palette
  // that is still being staged.
  Timer {
    id: reloadTimer
    interval: 250
    repeat: false
    onTriggered: root.reloadThemeFiles()
  }

  FileView {
    id: currentDirWatcher
    path: root.statePath
    watchChanges: true
    printErrors: false
    onFileChanged: root.scheduleReload()
  }

  FileView {
    id: themeNameFile
    path: root.themeNamePath
    watchChanges: true
    printErrors: false
    onLoaded: root.scheduleRefresh()
    onFileChanged: root.scheduleReload()
    onLoadFailed: root.scheduleRefresh()
  }

  FileView {
    id: colorsFile
    path: root.colorsPath
    watchChanges: true
    printErrors: false
    onLoaded: root.scheduleRefresh()
    onFileChanged: root.scheduleReload()
    onLoadFailed: root.scheduleRefresh()
  }

  FileView {
    id: generationFile
    path: root.generationPath
    watchChanges: true
    printErrors: false
    onLoaded: root.scheduleRefresh()
    onFileChanged: root.scheduleReload()
  }

  FileView {
    id: configFile
    path: root.configPath
    watchChanges: true
    printErrors: false
    onLoaded: root.applyConfig(text())
    onLoadFailed: root.applyConfig("")
    onFileChanged: reload()
  }

  Process {
    id: writeProcess
    onExited: function(exitCode) {
      if (exitCode !== 0) {
        root.failWrite("Could not write " + root.activeOutputPath)
        return
      }
      root.finishWrite()
    }
  }

  Component.onCompleted: root.scheduleRefresh()
}
