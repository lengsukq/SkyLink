.PHONY: dev run build

# 开发：热重载（无需单独安装 air）
dev:
	go run github.com/air-verse/air@latest

# 普通启动
run:
	go run ./cmd/server

# 编译
build:
	go build -o skylink ./cmd/server
