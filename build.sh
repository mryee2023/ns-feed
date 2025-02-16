#!/bin/bash

# 默认值设置
APP_NAME="ns-feed-bot"
DOCKER_REPO="your-dockerhub-username"  # 替换为你的 Docker Hub 用户名
VERSION=$(git describe --tags --always || echo "latest")
BUILD_DOCKER=false

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --docker)
            BUILD_DOCKER=true
            shift
            ;;
        --repo=*)
            DOCKER_REPO="${1#*=}"
            shift
            ;;
        --version=*)
            VERSION="${1#*=}"
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# 检查是否安装了必要的交叉编译工具
check_cross_compile_tools() {
    if ! command -v docker &> /dev/null; then
        echo "Error: Docker is required for cross-compilation"
        exit 1
    fi
}

# 主构建流程
echo "Starting multi-platform build..."

# 确保 bin 目录存在
mkdir -p bin

# 检查交叉编译工具
check_cross_compile_tools

# 构建各平台版本
for platform in "linux/amd64" "darwin/amd64" "windows/amd64"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    OUTPUT_NAME="${APP_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    docker run --rm \
        -v $(pwd):/go/src/app \
        -w /go/src/app \
        -e CGO_ENABLED=0 \
        -e GOOS=$GOOS \
        -e GOARCH=$GOARCH \
        golang:1.22 \
        go build -o ./bin/$OUTPUT_NAME ./src/main.go
done

echo "Build completed!"

# Docker 相关操作
if [ "$BUILD_DOCKER" = true ]; then
    echo "Building Docker image..."
    docker build -t ${APP_NAME}:${VERSION} .
    
    echo "Tagging Docker image..."
    docker tag ${APP_NAME}:${VERSION} ${DOCKER_REPO}/${APP_NAME}:${VERSION}
    docker tag ${APP_NAME}:${VERSION} ${DOCKER_REPO}/${APP_NAME}:latest
    
    echo "Pushing Docker image..."
    docker push ${DOCKER_REPO}/${APP_NAME}:${VERSION}
    docker push ${DOCKER_REPO}/${APP_NAME}:latest
    
    echo "Docker image pushed successfully!"
fi