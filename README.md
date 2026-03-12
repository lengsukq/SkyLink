# SkyLink

单隧道分流 + Cloudflare DNS 管理 + DDNS，带 Web 管理界面。适用于家里服务器仅有一条樱花 frp 隧道时，按域名将流量转发到不同本地项目。

## 功能

- **反向代理**：按 Host（如 `xx.yyy.com`、`x1.yyy.com`）转发到不同本地端口
- **一键映射**：添加映射时可选同时创建 Cloudflare CNAME 记录
- **Cloudflare**：管理 Zone、DNS 记录（A/CNAME/AAAA）
- **DDNS**：定时将当前公网 IP 更新到指定 CF A 记录
- **SQLite**：映射与 DDNS 配置持久化

## 本地开发

### 依赖

- Go 1.21+
- Node 18+（仅构建前端时需要）

### 后端

```bash
go mod tidy
go run ./cmd/server
```

默认：反代 `:18080`，管理 `:19080`，数据库 `./data/skylink.db`。

### 前端（可选，用于开发时热更新）

```bash
cd web && npm install && npm run dev
```

浏览器访问 `http://localhost:5173`，Vite 会代理 `/api` 到 `:19080`。

### 打包含前端的二进制

```bash
cd web && npm ci && npm run build
go build -o skylink ./cmd/server
```

## Docker

```bash
docker compose up -d
```

- 反代：`http://localhost:18080`
- 管理界面：`http://localhost:19080`

### 管理端登录

- **首次启动**：服务日志会打印一次随机密码（仅第一次生成时打印）。
- **登录地址**：`http://<host>:19080/#/login`
- **修改密码**：登录后在「设置」页面修改；新密码会持久化到 SQLite（挂载 `./data` 目录即可持久化）。
- **鉴权方式**：管理 API 需要 `Authorization: Bearer <密码>`。

## GitHub Actions（Docker 镜像）

仓库内置 Workflow：push 到 `main` 或手动触发时会构建并推送到 GHCR。

- **镜像地址**：`ghcr.io/<owner>/<repo>`
- **版本号（tag）**：`v<github.run_number>`（即 GitHub 构建号），同时推送 `latest`
- **权限**：使用 `GITHUB_TOKEN` 推送到 GitHub Packages（需要 `packages: write`）

同一 Workflow 也会创建一个对应版本的 **GitHub Release**（tag 同上），并上传可下载的二进制：

- `skylink_linux_amd64`
- `skylink_linux_arm64`
- `SHA256SUMS`

### 环境变量

| 变量 | 说明 |
|------|------|
| `SKYLINK_PROXY_PORT` | 反代监听端口，默认 18080 |
| `SKYLINK_ADMIN_PORT` | 管理 API/GUI 端口，默认 19080 |
| `SKYLINK_DB_PATH` | SQLite 路径，默认 `./data/skylink.db` |
| `CF_API_TOKEN` | Cloudflare API Token（Zone.DNS Edit） |
| `CF_ZONE_ID` | 默认 Zone ID（可选） |

### 与樱花 Frp 配合

1. 樱花 frp 创建一条隧道：将 80（及可选 443）转发到本机 SkyLink 反代端口（如 18080）。
2. 在 SkyLink 管理界面添加映射：域名 `xx.yyy.com` → `http://127.0.0.1:3000` 等。
3. 在 Cloudflare 将域名 CNAME 到樱花出口域名，或使用「一键映射」自动创建 CNAME。

## 配置文件（可选）

可通过 `-config config.yaml` 指定 YAML 配置，例如：

```yaml
app:
  proxy_port: 80
  admin_port: 8080
  db_path: ./data/skylink.db

cloudflare:
  api_token: ""
  zone_id: ""
```

敏感信息建议用环境变量覆盖。
