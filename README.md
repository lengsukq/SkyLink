# SkyLink

单隧道分流 + Cloudflare DNS 管理 + DDNS（IPv4/IPv6）+ 可选 EasyTier Mesh，一体化 Web 管理面板。

适用于家庭服务器只有一个公网入口（如 Frp/内网穿透）时，按域名把流量转发到不同内网服务，同时统一管理 DNS 和动态解析。

## 核心能力

| 模块 | 能力 |
|------|------|
| 反向代理 | 按请求 Host 转发到不同后端服务（例如 `app.example.com -> http://127.0.0.1:3000`） |
| 一键映射 | 创建映射时可联动创建 Cloudflare DNS 记录 |
| Cloudflare 中心 | 多账号、分 Zone 管理 DNS 记录（A/AAAA/CNAME/TXT/MX 等） |
| DDNS | 定时同步公网 IPv4/IPv6 到 Cloudflare 记录 |
| EasyTier（可选） | 管理 mesh 配置、版本、守护进程状态，映射可直连 mesh 节点 |
| 文件服务 | Drive/WebDAV/SMB 映射与运行状态管理 |
| 管理安全 | 基于 Bearer Token 的管理 API 鉴权，管理员密码可持久化 |

## 技术栈

- 后端：Go 1.21、Gin、SQLite（`modernc.org/sqlite`）
- 前端：Vue 3、Vite、TypeScript、Pinia、Naive UI、Tailwind CSS
- 部署：Docker 多阶段构建 + `docker compose`
- CI/CD：GitHub Actions（镜像 + Release 产物）

## 项目结构

```text
SkyLink/
├─ cmd/server/              # 服务入口
├─ internal/                # 后端业务模块（api/proxy/ddns/cf/easytier/drive/smb/webdav...）
├─ web/                     # 前端管理界面（Vue3 + Vite）
├─ static/                  # 前端构建产物嵌入目录（go:embed）
├─ scripts/                 # 构建/打包脚本（含 Windows）
├─ data/                    # 运行时数据（SQLite、easytier env/runtime 等）
├─ Dockerfile
├─ docker-compose.yml
├─ Makefile
└─ config.example.yaml
```

## 快速开始

### 1) 本地运行后端

```bash
go mod tidy
go run ./cmd/server
```

默认端口：

- 反向代理入口：`http://127.0.0.1:18080`
- 管理界面/API：`http://127.0.0.1:19080`
- 数据库：`./data/skylink.db`

### 2) 本地开发（热更新）

```bash
make dev
```

该命令通过 `.air.toml` 启动 Air 热更新（无需单独安装 Air）。

### 3) 前端开发（可选）

```bash
cd web
npm ci
npm run dev
```

访问 `http://localhost:5173`，开发代理会把 `/api` 转发到 `:19080`。

### 4) 构建可发布二进制（含前端）

```bash
cd web
npm ci
npm run build
cd ..
go build -o skylink ./cmd/server
```

## 常用命令

### Makefile

- `make dev`：Go 热更新开发
- `make run`：直接运行服务
- `make build`：构建后端二进制
- `make sync-web`：构建并同步前端到 `static/web/dist`
- `make package-win`：Windows 打包脚本

### 前端脚本（`web/package.json`）

- `npm run dev`：本地开发
- `npm run build`：生产构建
- `npm run preview`：预览构建结果
- `npm run test`：Vitest
- `npm run test:coverage`：Vitest 覆盖率报告（v8）
- `npm run type-check`：TS/Vue 类型检查
- `npm run check:all`：完整检查（类型、测试、构建）

> 前端包管理器建议统一使用 `npm`（项目 CI 也按 `npm ci` 执行）。

### 后端测试

```bash
go test ./...
```

## Docker 部署

### 方式 A：`docker run`

```bash
docker pull queensu/skylink:latest
docker run -d \
  --name skylink \
  --restart unless-stopped \
  -p 18080:18080 \
  -p 19080:19080 \
  -v "$(pwd)/data:/data" \
  -e SKYLINK_DB_PATH=/data/skylink.db \
  queensu/skylink:latest
```

### 方式 B：`docker compose`

项目已提供 `docker-compose.yml`，直接启动：

```bash
docker compose up -d
```

## 登录与鉴权

- 首次启动会在日志打印一次随机管理员密码（仅首次生成时输出）。
- 登录地址：`http://<host>:19080/#/login`
- 管理 API 需携带请求头：`Authorization: Bearer <token>`
- 可用 `SKYLINK_ADMIN_PASSWORD` 固定管理员密码（与数据库持久化密码并存）。

## 配置说明

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SKYLINK_PROXY_PORT` | `18080` | 反代监听端口 |
| `SKYLINK_ADMIN_PORT` | `19080` | 管理 API/前端端口 |
| `SKYLINK_DB_PATH` | `./data/skylink.db` | SQLite 文件路径 |
| `SKYLINK_REQUIRE_ADMIN` | `false` | 是否启用 Windows 非管理员启动时的提权重开 |
| `SKYLINK_ADMIN_PASSWORD` | 空 | 固定管理员密码 |
| `SKYLINK_EASYTIER_RPC` | `127.0.0.1:15888` | EasyTier RPC 地址 |
| `SKYLINK_EASYTIER_ENV_FILE` | `./data/easytier.env` | EasyTier env 文件路径 |
| `SKYLINK_EASYTIER_DAEMON_ENABLED` | `true` | 启用内置 Daemon 管理 |
| `SKYLINK_EASYTIER_DAEMON_PATH` | `easytier-core` | EasyTier daemon 可执行文件路径 |
| `SKYLINK_EASYTIER_RUNTIME_DIR` | `./data/easytier-bin` | EasyTier 运行时缓存目录 |
| `SKYLINK_EASYTIER_DAEMON_CAPTURE_OUTPUT` | `true` | 是否捕获 Daemon 输出（Windows 设为 `0/false` 可弹控制台调试） |

### YAML 配置（可选）

通过 `-config config.yaml` 指定，示例见 `config.example.yaml`：

```yaml
app:
  proxy_port: 18080
  admin_port: 19080
  db_path: ./data/skylink.db

easytier:
  rpc_address: "127.0.0.1:15888"
  enabled: true
  daemon_enabled: true
  daemon_path: ""
  runtime_dir: "./data/easytier-bin"
```

> 建议：敏感项优先使用环境变量，不写入配置文件。

## EasyTier 使用提示（可选）

- SkyLink 支持多 profile 管理 EasyTier 配置，每个 profile 可独立启停和查看状态。
- 修改网络配置或版本后，需要重启 EasyTier Daemon 才会生效。
- 若看到 `tun error Operation not permitted`，通常是 TUN 权限不足：
  - 裸机运行请使用具备权限的账户；
  - Docker 需添加 `--cap-add=NET_ADMIN --device /dev/net/tun`。

Docker 权限示例：

```yaml
services:
  skylink:
    image: queensu/skylink:latest
    cap_add:
      - NET_ADMIN
    devices:
      - /dev/net/tun:/dev/net/tun
```

## 与 Frp 场景配合（典型用法）

1. 把 Frp 外部流量转发到 SkyLink `18080`。
2. 在 SkyLink 中添加域名映射到内网服务。
3. 在 Cloudflare 设置 CNAME，或用“一键映射”自动创建记录。
4. 需要动态公网 IP 时，在 DDNS 页面配置对应 A/AAAA 记录。

## CI/CD 与发布

仓库包含 GitHub Actions 发布工作流（`.github/workflows/docker-image.yml`）：

- 支持按 tag（`v*`）或手动触发构建
- 推送多架构镜像到 GHCR / Docker Hub
- 生成 Release 附件（Linux `amd64/arm64` 二进制与校验和）

详细职责拆分与维护约定见 [docs/ci.md](docs/ci.md)。

质量门工作流（`.github/workflows/quality-gates.yml`）：

- PR 与 `main` 分支提交会执行后端 `go test` 覆盖率采集与阈值检查
- 前端执行 `npm run type-check`、`npm run test`、`npm run test:coverage`、`npm run build`
- 会上传前后端覆盖率产物，便于回归对比

## 贡献与维护

- 贡献与质量门细则见 [CONTRIBUTING.md](CONTRIBUTING.md)

## 许可证

请根据仓库实际 License 文件补充。
