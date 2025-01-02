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

# 确保 bin 目录存在
mkdir -p bin

# 简化的构建函数
build_binary() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="${APP_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi

    echo "Building for $GOOS/$GOARCH..."
    
    # 对于所有平台，我们都使用 CGO_ENABLED=0 来简化构建过程
    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o ./bin/$OUTPUT_NAME ./src/main.go
    
    if [ $? -eq 0 ]; then
        echo "Successfully built $OUTPUT_NAME"
    else
        echo "Failed to build $OUTPUT_NAME"
        exit 1
    fi
}

# 主构建流程
echo "Starting multi-platform build..."

# 构建各平台版本
build_binary "linux" "amd64"
build_binary "linux" "arm64"
build_binary "darwin" "amd64"
build_binary "darwin" "arm64"
build_binary "windows" "amd64"

# 如果需要构建 Docker 镜像
if [ "$BUILD_DOCKER" = true ]; then
    echo "Building Docker image..."
    docker build -t ${DOCKER_REPO}/${APP_NAME}:${VERSION} \
        --build-arg APP_NAME=${APP_NAME} \
        --build-arg VERSION=${VERSION} \
        -f Dockerfile .
    
    echo "Pushing Docker image..."
    docker push ${DOCKER_REPO}/${APP_NAME}:${VERSION}
    docker tag ${DOCKER_REPO}/${APP_NAME}:${VERSION} ${DOCKER_REPO}/${APP_NAME}:latest
    docker push ${DOCKER_REPO}/${APP_NAME}:latest
fi

echo "Build process completed!"