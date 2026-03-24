package static

import "embed"

// FS embeds the built frontend.
//
// Build first: cd web && npm install && npm run build
//
//go:embed web/dist
var FS embed.FS
