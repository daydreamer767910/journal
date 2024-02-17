# syntax=docker/dockerfile:1

FROM golang:1.21-alpine as builder

ARG APP_VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# 更新包列表并安装 FFmpeg 开发库
RUN apk update && \
    apk add --no-cache ffmpeg-dev libc-dev pkgconf make gcc


WORKDIR /build
COPY go.* ./

RUN go mod download

COPY *.go ./
COPY Makefile ./
COPY router /build/router
COPY handler /build/handler
COPY model /build/model
COPY store /build/store
COPY util /build/util
COPY web /build/web
COPY templates /build/templates

RUN CGO_ENABLED=1 GOOS=linux make

# 第二个阶段 - 生产阶段
#FROM alpine:3.16
FROM alpine:latest

# 更新包列表并安装 FFmpeg
RUN apk update && \
    apk add --no-cache ffmpeg libc6-compat

# 设置运行时环境变量
ENV LD_LIBRARY_PATH /usr/lib
# 复制构建阶段生成的文件到生产镜像中
WORKDIR /app

COPY --from=builder /build/journal .

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 5000/tcp

# Run
CMD ["./journal"]