package static

import "embed"

// FS embeds the built frontend.
// Windows: scripts\sync-web-to-static.ps1（或 dev-windows.cmd）会 npm run build 并同步到 web/dist，再 touch 本文件以触发 Air 重新嵌入。
//
//go:embed web/dist
var FS embed.FS
