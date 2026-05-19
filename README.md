# feed-puller

单用户自用的 RSS 自动下载管理工具。后端使用 Go，前端使用 React + TypeScript，数据存储连接外部 MySQL，下载任务提交到外部 aria2 JSON-RPC。

## 功能

- 账号密码登录，启动时通过环境变量初始化管理员。
- 飞书作为备用登录：管理员登录后在设置页绑定飞书账号。
- 每个 RSS 订阅可配置名称、订阅地址、拉取计划（固定间隔分钟数，或可选标准五字段 crontab：分 时 日 月 周）、crontab 所用 **IANA 时区**（如 `Asia/Shanghai`，空则用 UTC）、保存路径、启用状态、是否使用代理、包含/排除关键字（每行一条 Go 正则，对标题、链接与下载地址联合匹配；排除优先，包含非空时须命中至少一条）。
- 全局 HTTP/HTTPS 代理只用于获取 RSS 内容，不用于 aria2 RPC，也不参与实际下载。
- 解析 RSS/Atom 标准字段，优先使用 `enclosure`，也支持条目链接、magnet、torrent URL。
- 条目按 `guid`、规范化链接、下载 URL 去重；已触发下载的条目不会重复进入队列。

## 环境变量

复制 `.env.example` 并按实际环境配置：

```bash
cp .env.example .env
```

必填项：

- `MYSQL_DSN`：外部 MySQL DSN，需要带 `parseTime=true`。
- `ADMIN_EMAIL` / `ADMIN_PASSWORD`：启动时初始化或更新管理员账号。
- `SESSION_SECRET`：至少 32 个字符。
- `ARIA2_RPC_URL`：外部 aria2 JSON-RPC 地址，例如 `http://127.0.0.1:6800/jsonrpc`。

飞书登录需要在飞书开放平台配置回调地址：

```text
${BASE_URL}/api/auth/feishu/callback
```

## 本地开发

安装依赖：

```bash
npm install
go mod download
```

前端开发服务：

```bash
npm run dev
```

后端服务：

```bash
set -a
. ./.env
set +a
go run ./cmd/feed-puller
```

生产构建：

```bash
npm run build
go build -o bin/feed-puller ./cmd/feed-puller
```

## Docker 运行

项目提供多阶段 `Dockerfile`，镜像内会构建前端并打包 Go 后端。Docker 支持只负责运行 feed-puller 应用本身，MySQL 和 aria2 仍按外部服务配置。

准备 `.env`：

```bash
cp .env.example .env
```

如果 MySQL 或 aria2 跑在宿主机上，容器内不能使用 `127.0.0.1` 访问宿主机服务。Docker Desktop 可改成：

```env
MYSQL_DSN=feed_puller:feed_puller@tcp(host.docker.internal:3306)/feed_puller?parseTime=true&loc=UTC
ARIA2_RPC_URL=http://host.docker.internal:6800/jsonrpc
BASE_URL=http://localhost:8080
```

构建镜像：

```bash
docker build -t feed-puller:local .
```

使用 Compose 启动：

```bash
docker compose up -d --build
```

停止：

```bash
docker compose down
```

## 验证

```bash
GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./...
npm test
npm run build
docker build -t feed-puller:local .
```

当前仓库的 Go 环境如果默认设置为 `linux/amd64`，在 macOS 上运行测试需要显式设置 `GOOS=darwin GOARCH=arm64`。

## API 概览

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `GET /api/auth/feishu/start`
- `GET /api/auth/feishu/callback`
- `GET/POST /api/subscriptions` — **新建订阅不会立即拉取 RSS**；首次内容依赖定时调度（按创建时间与间隔/crontab 计算）或 `refresh` 手动拉取。
- `GET/PUT/DELETE /api/subscriptions/{id}`
- `POST /api/subscriptions/{id}/refresh` — 拉取 RSS 并写入条目，响应 `{ "items": [...] }`（条目含可选 `content_length` 字节数）；**不会**自动提交 aria2 下载。
- `GET /api/items`
- `POST /api/items/{id}/download` — 将单条条目提交给 aria2（等同于队列中的单条处理）。
- `GET /api/downloads`
- `GET/PUT /api/settings/proxy`
- `GET/DELETE /api/settings/feishu-binding`
