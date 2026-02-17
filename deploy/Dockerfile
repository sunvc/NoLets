# 第一阶段：构建阶段
FROM golang:1.25.0-alpine AS builder

# 设置工作目录
WORKDIR /app

#ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0

COPY . .

# 接收版本信息作为构建参数
ARG VERSION
ARG BUILD_DATE
ARG COMMIT_ID

RUN go mod tidy && go mod verify && go mod download

RUN echo "VERSION=${VERSION} BUILD_DATE=${BUILD_DATE} COMMIT_ID=${COMMIT_ID}"

RUN go build -v -x -tags "sonic avx" -ldflags="-X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.commitID=${COMMIT_ID}" -o main main.go
FROM alpine

WORKDIR /app

ARG TIMEZONE=utc

RUN set -ex \
#    && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk update \
    && apk add ca-certificates \
    && apk add --no-cache tzdata\
    && rm -rf /var/lib/apk/lists/*

COPY --from=builder /app/main .

VOLUME /data

EXPOSE 8080

ENTRYPOINT ["./main"]


