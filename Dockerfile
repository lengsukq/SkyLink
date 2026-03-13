# Stage 1: build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json ./
RUN npm install
COPY web/ .
RUN npm run build

# Stage 2: build Go binary (with embedded frontend)
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
COPY --from=frontend /app/web/dist ./static/web/dist
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /skylink ./cmd/server

# Stage 2b: EasyTier CLI（与 daemon 同版本，用于 status/version；若镜像中无则生成空占位，EasyTier 页状态会不可用）
FROM easytier/easytier:latest AS easytier
RUN F=$(find / -name 'easytier-cli' -type f 2>/dev/null | head -1) && if [ -n "$F" ]; then cp "$F" /out-easytier-cli; else touch /out-easytier-cli; fi

# Stage 3: minimal runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /skylink /skylink
COPY --from=easytier /out-easytier-cli /usr/local/bin/easytier-cli
WORKDIR /data
ENV SKYLINK_DB_PATH=/data/skylink.db
EXPOSE 18080 19080
ENTRYPOINT ["/skylink"]
CMD []
