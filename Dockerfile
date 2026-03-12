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

# Stage 3: minimal runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /skylink /skylink
WORKDIR /data
ENV SKYLINK_DB_PATH=/data/skylink.db
EXPOSE 18080 19080
ENTRYPOINT ["/skylink"]
CMD []
