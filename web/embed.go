// Package web embeds the browser application so Hearthlight ships as one binary.
package web

import "embed"

//go:embed index.html app.js style.css themes.json themes.js theme-boot.js
var Files embed.FS
