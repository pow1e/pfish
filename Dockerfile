# 使用指定版本的 Go 构建镜像
FROM golang:1.20-alpine AS builder

LABEL stage=gobuilder

# 设置环境变量
ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

# 更换为阿里云的apk镜像源，避免网络问题
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装时区依赖
RUN apk update --no-cache && apk add --no-cache tzdata

# 设置工作目录
WORKDIR /build

# 拷贝 go.mod 和 go.sum 文件
ADD go.mod ./
ADD go.sum ./

# 下载依赖并整理
RUN go mod download
RUN go mod tidy

# 拷贝项目源代码
COPY . .

# 打印 go.mod 内容，检查依赖是否同步
RUN cat go.mod

# 构建 Go 程序，指定正确的路径
RUN go build  -o /app/fish server.go

# 运行时镜像仍然是 golang:alpine
FROM golang:1.20-alpine

# 安装必要的依赖
RUN apk update && apk add --no-cache ca-certificates tzdata

# 拷贝构建好的二进制文件
COPY --from=builder /app/fish /app/fish

# 拷贝配置文件
COPY config.yaml /app/config.yaml

# 拷贝go.mod go.sum api 用于生成agent
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY api /app/api

# 拷贝garble
COPY garble /app/garble

RUN chmod +x /app/garble


# 设置工作目录
WORKDIR /app

# 暴露端口
EXPOSE 50001 50002

# 启动应用
CMD ["./fish"]

