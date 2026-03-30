.PHONY: dev run build package-win sync-web

# Windows: .\scripts\sync-web-to-static.ps1 — 构建 web 并复制到 static/web/dist（go:embed），开发前可单独执行
sync-web:
	powershell -ExecutionPolicy Bypass -File .\scripts\sync-web-to-static.ps1

# 开发：热重载（无需单独安装 air）
dev:
	go run github.com/air-verse/air@latest

# 普通启动
run:
	go run ./cmd/server

# 编译
build:
	go build -o skylink ./cmd/server

# Windows 一键打包（前端 + 内嵌静态 + exe）
package-win:
	powershell -ExecutionPolicy Bypass -File .\scripts\package-windows.ps1
