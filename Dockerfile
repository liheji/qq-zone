# Stage 1: Build web server
FROM golang:1.23-alpine as go-builder
LABEL stage=go-builder
WORKDIR /app/

# Go 依赖下载
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 构建二进制文件
COPY ./ ./
RUN go build -o qq-zone main.go


# Stage 12 Builder static
FROM node:20-alpine AS node-builder
LABEL stage=node-builder
WORKDIR /app

# npm 依赖下载
COPY ./frontend/package.json ./
RUN npm config set registry https://registry.npmmirror.com
RUN npm i

# 构建前端
COPY ./frontend .
RUN npm run build


# Stage 3: Get Result
FROM debian:stable-slim
LABEL MAINTAINER="930617673@qq.com"
WORKDIR /app

# 安装运行环境
RUN sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list.d/debian.sources
RUN apt update && \
    apt install -y --no-install-recommends tzdata libimage-exiftool-perl  ca-certificates && \
    apt clean && \
    rm -rf /var/lib/apt/lists/*

# 复制二进制文件
RUN mkdir /app/storage && mkdir /app/templates
COPY --from=go-builder /app/qq-zone ./qq-zone
COPY --from=node-builder /app/dist ./templates
RUN chmod +x /app/qq-zone

# 配置环境变量
ENV PUID=0 PGID=0 UMASK=022

EXPOSE 9000
ENTRYPOINT ["/app/qq-zone"]


