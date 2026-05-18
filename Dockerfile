# syntax=docker/dockerfile:1.7

FROM node:22-bookworm-slim AS frontend
WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts ./
COPY web ./web
RUN npm run build

FROM golang:1.24-bookworm AS backend
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/feed-puller ./cmd/feed-puller

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

ENV PORT=8080
ENV STATIC_DIR=/app/web/dist

COPY --from=backend /out/feed-puller /app/feed-puller
COPY --from=frontend /src/web/dist /app/web/dist

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/feed-puller"]

