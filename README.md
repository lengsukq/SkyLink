# SkyLink

单隧道分流 + Cloudflare DNS 管理 + DDNS（IPv4/IPv6），带 Web 管理界面。适用于家庭服务器仅有一条隧道（如樱花 Frp）时，按域名将流量转发到不同本地服务，并统一管理 DNS 与动态解析。

## 功能概览

| 功能 | 说明 |
|------|------|
| **反向代理** | 按请求 Host（如 `app.example.com`）转发到不同本地后端（如 `http://127.0.0.1:3000`） |
| **一键映射** | 添加映射时可同时创建 Cloudflare CNAME 记录，并可使用设置中的默认 CNAME 目标（如 Frp 出口域名） |
| **Cloudflare** | 多账号支持；在选定账号下管理 Zone、DNS 记录（A / AAAA / CNAME / TXT / MX 等） |
| **DDNS** | 定时将当前公网 **IPv4 / IPv6** 更新到指定 Cloudflare A 或 AAAA 记录，按账号隔离 |
| **SQLite** | 映射、DDNS 配置、Cloudflare 账号与设置持久化 |

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

### 前端（可选，开发时热更新）

```bash
cd web && npm install && npm run dev
```

浏览器访问 `http://localhost:5173`，Vite 会将 `/api` 代理到 `:19080`。

### 打包含前端的二进制

```bash
cd web && npm ci && npm run build
go build -o skylink ./cmd/server
```

## Docker 部署

### 方式 A：docker run

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

### 方式 B：docker compose

示例 `docker-compose.yml`：

```yaml
services:
  skylink:
    image: queensu/skylink:latest
    ports:
      - "18080:18080"
      - "19080:19080"
    volumes:
      - ./data:/data
    environment:
      - SKYLINK_DB_PATH=/data/skylink.db
    restart: unless-stopped
```

启动：

```bash
docker compose up -d
```

- **反代入口**：`http://<host>:18080`
- **管理界面**：`http://<host>:19080`

## 管理端登录与鉴权

- **首次启动**：服务日志会打印一次随机管理员密码（仅首次生成时打印）。
- **登录地址**：`http://<host>:19080/#/login`
- **修改密码**：登录后在「设置」页修改；密码持久化到 SQLite（挂载 `./data` 即可持久化）。
- **鉴权**：管理 API 需在请求头中携带 `Authorization: Bearer <密码>`。

可选环境变量：

- `SKYLINK_ADMIN_PASSWORD`：固定管理员密码（不写入数据库，与随机/持久化密码并存）。

## 环境变量

| 变量 | 说明 |
|------|------|
| `SKYLINK_PROXY_PORT` | 反代监听端口，默认 18080 |
| `SKYLINK_ADMIN_PORT` | 管理 API / 前端端口，默认 19080 |
| `SKYLINK_DB_PATH` | SQLite 路径，默认 `./data/skylink.db` |

## 配置文件（可选）

通过 `-config config.yaml` 指定 YAML 配置，例如：

```yaml
app:
  proxy_port: 18080
  admin_port: 19080
  db_path: ./data/skylink.db
```

敏感信息建议用环境变量覆盖。Cloudflare 与 DDNS 使用管理界面中配置的多账号与 DDNS 条目，无需在配置文件中填写 API Token。

## 与樱花 Frp 配合

1. 在樱花 Frp 创建一条隧道：将 80（及可选 443）转发到本机 SkyLink 反代端口（如 18080）。
2. 在 SkyLink 管理界面「映射」页添加映射：域名 `xx.yyy.com` → 后端 `http://127.0.0.1:3000` 等。
3. 在 Cloudflare 将域名 CNAME 到樱花出口域名，或使用「一键映射」在添加映射时自动创建 CNAME。

## DDNS 说明

- 支持 **A（IPv4）** 与 **AAAA（IPv6）** 记录；每条 DDNS 配置对应一条记录类型。
- 公网 IP 通过公共接口获取（IPv4 / IPv6 分别有多个备用源）；后台按分钟轮询并更新已启用的配置。
- DDNS 配置与 Cloudflare 多账号绑定：在管理界面选择当前 CF 账号后，DDNS 列表与新增均在该账号下生效，更新时使用对应账号的 API 更新对应 Zone 下的记录。

## GitHub Actions（Docker 镜像与 Release）

仓库内 Workflow：推送到 `main` 或手动触发时构建并推送镜像到 GHCR，并创建 GitHub Release。

- **镜像**：`ghcr.io/<owner>/<repo>`
- **版本 tag**：`v<github.run_number>`，同时推送 `latest`
- **Release 附件**：`skylink_linux_amd64`、`skylink_linux_arm64`、`SHA256SUMS`

需为仓库配置 `packages: write` 权限。
