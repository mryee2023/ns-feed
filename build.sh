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

PLATFORMS=("linux/amd64" "windows/amd64" "darwin/amd64")

# 构建多平台二进制文件
for PLATFORM in "${PLATFORMS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "$PLATFORM"
    OUTPUT_NAME="${APP_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi
    if [ "$GOOS" == "darwin" ] && [ "$(uname)" == "Darwin" ]; then
        # 本地 Mac 编译启用 CGO
        echo "Building for $GOOS/$GOARCH with CGO enabled..."
        env CGO_ENABLED=1 GOOS=$GOOS GOARCH=$GOARCH go build -o ./bin/$OUTPUT_NAME ./src/main.go
    else
        # 跨平台编译禁用 CGO
        echo "Building for $GOOS/$GOARCH with CGO disabled..."
        env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o ./bin/$OUTPUT_NAME ./src/main.go
    fi
done

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