# syntax=docker/dockerfile:1.7
# 构建 amd64 镜像: docker build --platform linux/amd64 -t feed-puller:amd64 .
# BuildKit 会根据 --platform 自动注入 TARGETOS / TARGETARCH

FROM --platform=$BUILDPLATFORM node:20-bookworm-slim AS frontend
WORKDIR /src

# esbuild postinstall 在 overlayfs / 部分 libuv 版本上会触发 ETXTBSY（Text file busy）。
ENV npm_config_foreground_scripts=true

COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts \
  && (npm rebuild esbuild || (sleep 2 && npm rebuild esbuild))

COPY tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts ./
COPY web ./web
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS backend
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

# 由 --platform 注入；未指定时默认 linux/amd64
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/feed-puller ./cmd/feed-puller

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

ENV PORT=8080
ENV STATIC_DIR=/app/web/dist

COPY --from=backend /out/feed-puller /app/feed-puller
COPY --from=frontend /src/web/dist /app/web/dist

EXPOSE 8080
# 运行时 UID/GID 由 docker run -u 或 compose 的 PUID/PGID 覆盖（镜像默认 1000:1000）
USER 0:0
ENTRYPOINT ["/app/feed-puller"]
