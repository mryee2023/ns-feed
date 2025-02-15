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

# 使用Docker进行跨平台编译
build_with_docker() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="${APP_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi

    echo "Building for $GOOS/$GOARCH using Docker..."
    
    # 创建临时Dockerfile用于构建
    cat > Dockerfile.build << EOF
FROM golang:1.21

RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    libc6-dev \
    gcc-aarch64-linux-gnu \
    g++-aarch64-linux-gnu \
    libc6-dev-arm64-cross

WORKDIR /build
COPY . .

EOF

    if [ "$GOOS" == "linux" ]; then
        # 使用特定的Docker镜像进行Linux构建
        docker run --rm \
            -v $(pwd):/src \
            -w /src \
            -e CGO_ENABLED=1 \
            -e GOOS=linux \
            -e GOARCH=amd64 \
            -e CC=x86_64-linux-gnu-gcc \
            -e CXX=x86_64-linux-gnu-g++ \
            golang:1.22 \
            bash -c "apt-get update && apt-get install -y gcc-x86-64-linux-gnu g++-x86-64-linux-gnu libc6-dev-amd64-cross && go build -o ./bin/$OUTPUT_NAME ./src/main.go"
    elif [ "$GOOS" == "darwin" ]; then
        # macOS构建使用本地环境
        echo "Building darwin binary locally..."
        env CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o ./bin/$OUTPUT_NAME ./src/main.go
    else
        # Windows构建禁用CGO
        echo "Building windows binary with CGO disabled..."
        env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/$OUTPUT_NAME ./src/main.go
    fi

    rm -f Dockerfile.build
}

# 主构建流程
echo "Starting multi-platform build..."

# 检查交叉编译工具
check_cross_compile_tools

# 构建各平台版本
build_with_docker "linux" "amd64"
build_with_docker "darwin" "amd64"
build_with_docker "windows" "amd64"

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