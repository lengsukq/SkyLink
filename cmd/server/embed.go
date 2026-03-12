package main

import "embed"

// 构建前请先执行: cd web && npm install && npm run build
//go:embed all:../web/dist
var StaticFS embed.FS
