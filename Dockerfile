# Stage 1: 构建现代版前端
FROM node:22-alpine AS frontend-builder
WORKDIR /app

# 先复制静态资源目录，构建时会覆盖 web/full/
COPY web ./web

WORKDIR /app/frontend

# 通过 npm 镜像安装 pnpm 并安装依赖
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN npm config set registry https://registry.npmmirror.com && \
    npm install -g pnpm && \
    pnpm config set registry https://registry.npmmirror.com && \
    pnpm install --frozen-lockfile

# 复制前端源码并构建
COPY frontend ./
RUN pnpm build

# Stage 2: 构建 Go 后端
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app/backend

# 使用国内 Go 模块代理，避免默认 proxy.golang.org 超时
ENV GOPROXY=https://goproxy.cn,direct

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /multitune-server ./cmd/server/main.go

# Stage 3: 运行时镜像
FROM alpine:3.21

RUN apk add --no-cache ca-certificates curl ffmpeg

# 复制后端二进制
COPY --from=backend-builder /multitune-server /usr/local/bin/multitune-server

# 复制完整静态资源（入口页 + 车机版 + 完整版构建产物）
COPY --from=frontend-builder /app/web /app/static

ENV STATIC_PATH=/app/static \
    DATA_PATH=/app/data \
    PORT=8080 \
    LOG_LEVEL=info

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/multitune-server"]
