# feed-puller

单用户自用的 RSS 自动下载管理工具。后端使用 Go，前端使用 React + TypeScript，数据存储连接外部 MySQL，下载任务提交到外部 aria2 JSON-RPC。

## 功能

- 账号密码登录，启动时通过环境变量初始化管理员。
- 飞书作为备用登录：管理员登录后在设置页扫码绑定飞书账号（无需跳转确认）。
- 每个 RSS 订阅可配置名称、订阅地址、**RSS 解析器**（`generic` 通用 / `mikan` 蜜柑计划）、拉取计划（固定间隔分钟数，或可选标准五字段 crontab：分 时 日 月 周）、crontab 所用 **IANA 时区**（如 `Asia/Shanghai`，空则用 UTC）、保存路径、启用状态、是否使用代理、包含/排除关键字（每行一条 Go 正则，对标题、链接与下载地址联合匹配；排除优先，包含非空时须命中至少一条）。
- **蜜柑计划解析器**：拉取与下载时将 Mikan 的 `.torrent` 链接转换为 magnet 再提交 aria2，避免仅下载种子文件后 BT 任务卡住。
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

可选项：

- `ARIA2_HOOK_SECRET`：启用 aria2 钩子推送通道（详见下文「aria2 钩子接入」），未设置时 `/api/downloads/aria2-hook` 一律 401。
- `PASSWORD_LOGIN_ENABLED`：是否允许账号密码登录，默认 `true`；设为 `false` 时 `POST /api/auth/login` 返回 403，前端隐藏密码登录表单（需配置飞书登录并完成绑定）。

前端 UI 采用 Glassmorphism 设计系统，详见 `design-system/MASTER.md`。可在「设置 → 外观」切换 **玻璃暗色** / **玻璃浅色** / **跟随系统**（偏好保存在浏览器 `localStorage`，键名 `feed-puller-theme`）。

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

**AI 重命名与文件权限**：重命名在 feed-puller 进程内直接操作磁盘文件。目录已映射仍报 `permission denied` 时，通常是 **PUID/PGID 与 aria2 写文件的用户不一致**（不是挂载缺失）。

在 Linux 上重命名需要**父目录写权限**（`w+x`），与文件本身是否只读无关。仅把 `PUID`/`PGID` 设为 `0` 在 macOS Docker 上通常无效：容器内 root 仍无法改写由宿主机用户属主的目录。

1. **`PUID` / `PGID`**：与 aria2 进程一致（`.env` 注入 compose 的 `user:`）。查看：`docker exec <aria2容器> id` 或对已下载文件 `ls -n` 看 uid。
2. **订阅 `download_dir`**：填写**容器内**可见路径（与 volumes 映射一致）。
3. **路径映射**（可选）：仅当 aria2 钩子返回宿主机路径、与容器内挂载路径**不同**时设置 `DOWNLOAD_PATH_HOST_PREFIX` / `DOWNLOAD_PATH_CONTAINER_PREFIX`；已做同路径映射（如 `/data:/data`）则留空。
4. **volumes**：在 `docker-compose.yml` 自行配置；项目不再强制添加默认挂载，避免与已有映射冲突。

重命名失败时，错误信息会附带进程 uid、文件/目录属主 uid、目录是否可写。

停止：

```bash
docker compose down
```

## 验证

```bash
GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./...
npm test
npm run check:contrast   # 浅色玻璃主题 WCAG 对比度 + styles.css 令牌同步
npm run check:prowlarr-layout   # Prowlarr 布局常量与 styles.css 回退值同步
npm run test:e2e -- e2e/glass-a11y.spec.ts
npm run build
docker build -t feed-puller:local .
```

当前仓库的 Go 环境如果默认设置为 `linux/amd64`，在 macOS 上运行测试需要显式设置 `GOOS=darwin GOARCH=arm64`。

## API 概览

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `GET /api/auth/options` — 登录方式开关（`password_login_enabled`、`feishu_login_enabled`）
- `GET /api/auth/feishu/login-url` — 获取飞书扫码登录地址（`goto` 供 SDK 使用）
- `GET /api/auth/feishu/login` — 跳转飞书 passport 授权页
- `GET /api/auth/feishu/start` — 兼容旧入口，重定向到 login
- `GET /api/auth/feishu/callback` — 飞书 OAuth 回调（iframe 内 postMessage，不整页跳转）
- `GET/POST /api/subscriptions` — **新建订阅不会立即拉取 RSS**；首次内容依赖定时调度（按创建时间与间隔/crontab 计算）或 `refresh` 手动拉取。
- `GET/PUT/DELETE /api/subscriptions/{id}`
- `POST /api/subscriptions/{id}/refresh` — 拉取 RSS 并写入条目，响应 `{ "items": [...] }`（条目含可选 `content_length` 字节数）；新条目状态为 `preview`（仅预览），**不会**自动提交 aria2 下载，需在预览弹窗中手动下载或批量标记为「未处理」后由调度器提交。
- `GET /api/items`
- `POST /api/items/{id}/download` — 将单条条目提交给 aria2（等同于队列中的单条处理）。
- `GET /api/downloads`
- `POST /api/downloads/aria2-hook` — aria2 钩子上报回调，**无 session**，鉴权使用 `Authorization: Bearer ${ARIA2_HOOK_SECRET}`。
- `GET/PUT /api/settings/proxy`
- `GET/DELETE /api/settings/feishu-binding`
- `GET /api/settings/feishu-bind-url` — 获取当前用户飞书绑定扫码地址

## aria2 钩子接入（可选，强烈推荐）

默认 scheduler 每分钟轮询 `aria2.tellStatus`，存在最多 60 秒延迟，且 aria2 重启 / 清理 `max-download-result` 后 GID 丢失时只能在「下载完成」页推迟显示。开启钩子后改为 aria2 主动 push，**秒级**进入「下载完成」列表，并能用钩子传入的真实文件路径直接做 AI 重命名。

步骤：

1. 在 `.env` 设置 `ARIA2_HOOK_SECRET=<随机字符串>`，重启服务。
2. 把 `scripts/aria2-hook.sh` 复制到能被 aria2 进程访问到的路径（例如 `/etc/aria2/feed-puller-hook.sh`），赋可执行权限。
3. 给脚本注入两个环境变量（在 systemd unit、docker `environment:` 或 aria2 启动脚本里）：
   - `FEED_PULLER_URL=http://feed-puller:8080`（aria2 容器/主机可达的地址）
   - `ARIA2_HOOK_SECRET=<与服务端一致>`
4. 在 `aria2.conf` 中追加（**每个钩子只能写一条命令**）：

   **若 `on-download-complete` 已占用（例如 P3TERX `clean.sh`）**，用项目自带的串联脚本，不要覆盖原有 clean：

   ```conf
   on-download-complete=/path/to/scripts/aria2-on-download-complete.sh
   on-bt-download-complete=/path/to/scripts/aria2-on-bt-download-complete.sh
   on-download-error=/path/to/scripts/aria2-hook.sh error
   on-download-stop=/path/to/scripts/aria2-hook.sh stop
   ```

   可通过环境变量覆盖 clean 路径：`ARIA2_CLEAN_SCRIPT=/Users/ushopal/Downloads/clean.sh`（此为脚本默认值）。

   **若没有其它 on-download-complete 脚本**，可简化为：

   ```conf
   on-download-complete=/path/to/scripts/aria2-hook.sh file-complete
   on-bt-download-complete=/path/to/scripts/aria2-hook.sh bt-complete
   ```

注意事项：

- 磁力/BT 会先完成 `[METADATA]` 占位文件并触发 `on-download-complete`，**必须用不同事件名**：`file-complete`（单文件，不写库）与 `bt-complete`（整任务完成）。旧版两钩子都传 `complete` 时服务端会查 `tellStatus` 兜底，但仍建议按上文区分。
- 磁力下载在元数据阶段与实体文件阶段 **aria2 GID 不同**（`followedBy` / `following`）。feed-puller 会自动把数据库中的 `aria2_gid` 切换到实体下载的 GID，并用该 GID 统计进度与结单；钩子若只打到新 GID，也会通过 `following` 反查任务。
- `on-download-complete` 与 `on-bt-download-complete` 职责不同：前者可多次触发（含元数据文件），后者表示整任务结束；**不要**把 `bt-complete` 写到 `on-download-complete` 上。
- 钩子和轮询并存：钩子失败/丢失时仍由 scheduler 兜底，端点幂等可多次调用。
- 未匹配到 gid（用户在 aria2 里手动添加的下载）端点返回 200 + `matched=false`，不会干扰脚本。
