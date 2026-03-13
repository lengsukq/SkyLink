# SkyLink

单隧道分流 + Cloudflare DNS 管理 + DDNS（IPv4/IPv6），带 Web 管理界面。适用于家庭服务器仅有一条隧道（如樱花 Frp）时，按域名将流量转发到不同本地服务，并统一管理 DNS 与动态解析。

## 功能概览

| 功能 | 说明 |
|------|------|
| **反向代理** | 按请求 Host（如 `app.example.com`）转发到不同本地后端（如 `http://127.0.0.1:3000`） |
| **一键映射** | 添加映射时可同时创建 Cloudflare CNAME 记录，并可使用设置中的默认 CNAME 目标（如 Frp 出口域名） |
| **Cloudflare** | 多账号支持；在选定账号下管理 Zone、DNS 记录（A / AAAA / CNAME / TXT / MX 等） |
| **DDNS** | 定时将当前公网 **IPv4 / IPv6** 更新到指定 Cloudflare A 或 AAAA 记录，按账号隔离 |
| **EasyTier** | 可选 mesh VPN：在界面配置网络/版本，查看节点与路由，映射时可选 mesh 内节点作为后端 |
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

**后端热更新（开发时）：** 在项目根目录执行 `make dev`，修改 Go 代码会自动重新编译并重启（通过 [Air](https://github.com/air-verse/air)，无需单独安装）。

```bash
make dev
```

若已安装 Air，也可直接运行 `air`。如需指定配置文件，在 `.air.toml` 中把 `full_bin` 改为 `"./tmp/main -config config.yaml"`。

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
| `SKYLINK_EASYTIER_RPC` | EasyTier RPC 地址（SkyLink 连接守护进程用），默认 `127.0.0.1:15888` |
| `SKYLINK_EASYTIER_ENV_FILE` | EasyTier env 文件路径（可选，用于与外部进程/容器共享配置），默认与 DB 同目录的 `easytier.env` |

## 配置文件（可选）

通过 `-config config.yaml` 指定 YAML 配置，例如（推荐开启守护进程模式）：

```yaml
app:
  proxy_port: 18080
  admin_port: 19080
  db_path: ./data/skylink.db

# 可选：EasyTier（推荐使用本机守护进程模式）
easytier:
  rpc_address: "127.0.0.1:15888"
  enabled: true
  daemon_enabled: true
  daemon_path: ""          # 可选，留空则从 PATH 或自动下载
  runtime_dir: "./data/easytier-bin"
  # env_file_path: "./data/easytier.env" # 可选：需要与外部进程/容器共享配置时使用
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

## EasyTier（可选）

可选启用 [EasyTier](https://easytier.rs/) mesh VPN，使多台设备组成虚拟内网，并在「映射」中将后端指向 mesh 内节点。

### 快速上手（推荐路径：本机守护进程模式）

1. 按 EasyTier 官方文档安装 CLI / Daemon（参见 `https://easytier.cn/guide/introduction.html` 及“安装 CLI”章节），确保 `easytier-cli` 与 `easytier-daemon` 在本机可用。
2. 在 SkyLink 根目录创建 `config.yaml`，按上文示例启用：
   - `easytier.enabled = true`
   - `easytier.daemon_enabled = true`
   - `easytier.rpc_address = "127.0.0.1:15888"`
3. 启动 SkyLink：`go run ./cmd/server -config config.yaml`（或使用已编译二进制）。
4. 打开管理界面 → **EasyTier** 页，在「网络配置」卡片中：
   - 填写 **网络名**、**网络密钥**、**初始节点（peers）**。这些字段分别对应 EasyTier 的 `network_identity.name`、`network_identity.secret` 与 `peers[].uri`，会写入 `ET_NETWORK_NAME`、`ET_NETWORK_SECRET`、`ET_PEERS`。
   - 根据需要设置主机名（`hostname`）、公网发现节点（`external-node`）、子网代理（`proxy-networks`）、是否开启 DHCP 以及 VPN Portal。
5. 点击「保存配置」。配置会写入内部数据库，并在需要时写入 `./data/easytier.env` 以供外部进程复用。
6. 在「状态」卡片中使用“启动 / 停止 / 重启”按钮控制 EasyTier 守护进程，并确认本机 mesh IP 与 peers / 路由是否正常。
7. 在「映射」添加/编辑时，可使用「从 mesh 选择」下拉本机或对等节点 IP，填入端口后自动生成后端地址（如 `http://10.144.144.2:3000`）。

### 版本与高级配置

- **配置版本**：在 EasyTier 页「EasyTier 版本」中填写目标版本（镜像 tag 语义，如 `v2.2.3` 或 `latest`）。保存后会写入 `EASYTIER_IMAGE_TAG`，同时作为 RuntimeDownloader 下载守护进程时使用的版本号。
- **当前运行版本**：状态区显示通过 `easytier-cli` 获取的当前版本；当运行版本与配置中的版本不一致时，通常表示已修改配置但尚未重启 EasyTier 守护进程。
- **检查更新**：点击「检查更新」拉取 GitHub 最新 release，可「使用此版本」填入配置；升级时在 EasyTier 页选择目标版本并保存，然后通过“重启 Daemon”或重启 SkyLink 进程完成升级。
- **扩展配置项（与官方配置的对应关系）**：
  - `hostname` → `ET_HOSTNAME`：节点主机名（可选）。
  - `external-node` → `ET_EXTERNAL_NODE`：公网发现节点地址，如 `tcp://public.easytier.cn:11010`，仅用于协助发现公网节点，非必填。
  - `proxy-networks` → `ET_PROXY_NETWORKS`：子网代理 CIDR 列表，如 `10.0.0.0/24`，可用逗号或换行分隔多个，对应 EasyTier `vpn_portal_config.proxy_networks`。
  - `dhcp` → `ET_DHCP=1`：启用后 IPv4 可留空，由 EasyTier 自动分配。
  - `vpn_portal` → `ET_VPN_PORTAL`：VPN Portal（WireGuard）配置地址，如 `wg://0.0.0.0:11013/10.14.14.0/24`。

### WireGuard / VPN Portal

1. 在 EasyTier 页高级配置中填写 `VPN Portal（WireGuard）`，例如 `wg://0.0.0.0:11013/10.14.14.0/24`，并保存配置。
2. 通过 EasyTier 页的“重启 Daemon”按钮或重新启动守护进程，使 VPN Portal 配置生效。
3. EasyTier 运行正常后，在管理界面 **EasyTier** 页的「WireGuard 客户端配置（VPN Portal）」卡片点击「获取配置」，即可看到 `easytier-cli vpn-portal` 返回的 WireGuard 客户端配置文本。
4. 点击「复制」按钮，将配置导入本地 WireGuard 客户端即可接入 mesh 网络。

### 重启与升级

- 修改配置或版本后，需重启 EasyTier 守护进程使配置生效（可在 EasyTier 页通过按钮完成）。
- 升级版本：在 EasyTier 页选择新版本并保存，然后重启守护进程。

### 故障排查（EasyTier）

- **状态页显示“EasyTier 当前未启用”**：请在 EasyTier 页开启「启用」开关并保存配置，然后启动 EasyTier 守护进程（可通过页面按钮或手动启动）。
- **状态页提示“无法获取 EasyTier 状态 / RPC 地址不可达”**：
  - 检查 `SKYLINK_EASYTIER_RPC` 或 UI 中的 RPC 地址是否指向正确的守护进程地址（默认 `127.0.0.1:15888`）。
  - 确认 EasyTier 守护进程已启动，且监听地址与 SkyLink 中配置一致。
  - 确认 `easytier-cli` 已安装并在 SkyLink 进程可见的 PATH 中，可以在同一环境下执行 `easytier-cli --version` 验证。
- **WireGuard 配置为空或获取失败**：
  - 确认在 EasyTier 高级配置中已填入正确的 `VPN Portal（WireGuard）` 地址，并已重启守护进程。
  - 确认 EasyTier 已成功加入目标网络（状态页上应有 peers / 路由）。

### 裸机 + 一站式 EasyTier（Daemon 模式）

在裸机部署或源码直跑场景下，推荐让 SkyLink 直接拉起并管理 EasyTier 守护进程，实现“一站式”启用。支持两种方式准备 EasyTier 二进制：

- **方式 A：自行安装 EasyTier**（兼容已有环境）：
  - 确保 `easytier-cli` 与 `easytier-daemon`（或 `easytier-core`）已安装并可在 PATH 中找到，或准备好二进制的绝对路径。
  - 通过环境变量或 YAML 显式指定：

    环境变量：

    ```bash
    export SKYLINK_EASYTIER_DAEMON_ENABLED=1
    export SKYLINK_EASYTIER_DAEMON_PATH=/usr/local/bin/easytier-daemon # 可选，默认从 PATH 查找
    ```

    YAML：

    ```yaml
    easytier:
      rpc_address: "127.0.0.1:15888"
      enabled: true
      daemon_enabled: true
      daemon_path: "/usr/local/bin/easytier-daemon" # 可选
    ```

- **方式 B：由 SkyLink 自动下载 EasyTier 运行时**（无需预装）：
  - 启用 daemon 模式但不设置 `daemon_path`，SkyLink 会使用内置的 RuntimeDownloader：
    - 根据 EasyTier 页中配置的版本（镜像 tag）和后端平台（linux/amd64、darwin/arm64 等），从 GitHub Releases 自动下载对应的 EasyTier 守护进程二进制；
    - 下载结果缓存到 `./data/easytier-bin/<version>/<os-arch>/`（可通过 `SKYLINK_EASYTIER_RUNTIME_DIR` 或 YAML 的 `easytier.runtime_dir` 覆盖）。
  - 在 Web 界面 **EasyTier** 页的「EasyTier 版本」下方，会显示：
    - 当前后端平台（如 `linux/amd64`）；
    - 「下载/更新 EasyTier 运行时」按钮：点击后会为当前平台按当前版本下载或更新 EasyTier 二进制。

3. 启动 SkyLink（裸机）：

```bash
go run ./cmd/server -config config.yaml
```

4. 在管理界面 **EasyTier** 页：
   - 像容器模式一样配置网络名、密钥、peers、版本等并点击「保存配置」；
   - 配置会写入 `./data/easytier.env`；
   - 若开启 daemon 模式并勾选「启用」，SkyLink 会尝试自动启动 `easytier-daemon`。
5. 在 **状态** 卡片中：
   - 可以看到 EasyTier 当前运行状态（版本、mesh IP、peers / 路由）；
   - 额外看到 Daemon 状态（运行中 / 未运行 / 上次启动错误），并可一键「启动 / 停止 / 重启」 EasyTier 守护进程。

> 提示：Daemon 模式是源码运行和裸机部署场景下的推荐方式；即便在其他场景中，也始终可以通过 `SKYLINK_EASYTIER_DAEMON_PATH` 显式覆盖自动下载的二进制路径。

## GitHub Actions（Docker 镜像与 Release）

仓库内 Workflow：推送到 `main` 或手动触发时构建并推送镜像到 GHCR，并创建 GitHub Release。

- **镜像**：`ghcr.io/<owner>/<repo>`
- **版本 tag**：`v<github.run_number>`，同时推送 `latest`
- **Release 附件**：`skylink_linux_amd64`、`skylink_linux_arm64`、`SHA256SUMS`

需为仓库配置 `packages: write` 权限。
