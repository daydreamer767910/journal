# syntax=docker/dockerfile:1

FROM golang:1.21 as builder

ARG APP_VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

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
COPY assets /build/assets
COPY templates /build/templates

RUN CGO_ENABLED=0 GOOS=linux make

# 第二个阶段 - 生产阶段
#FROM alpine:3.16
FROM alpine:latest

# 更新包列表并安装 FFmpeg
RUN apk update && \
    apk add --no-cache ffmpeg
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