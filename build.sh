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
        echo "Warning: Docker is not installed. Will try to build without Docker."
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

    echo "Building for $GOOS/$GOARCH..."

    # 确保 bin 目录存在
    mkdir -p bin
    
    # 检查是否有Docker
    if command -v docker &> /dev/null; then
        if [ "$GOOS" == "linux" ]; then
            echo "Building Linux binary using Docker..."
            # 使用更简单的Docker命令，避免依赖特定的交叉编译工具
            docker run --rm \
                -v $(pwd):/app \
                -w /app \
                -e CGO_ENABLED=1 \
                -e GOOS=linux \
                -e GOARCH=amd64 \
                golang:1.21 \
                bash -c "apt-get update && apt-get install -y gcc libc6-dev && go build -o ./bin/$OUTPUT_NAME ./src/main.go"
        elif [ "$GOOS" == "darwin" ]; then
            echo "Building macOS binary using Docker..."
            # 对于macOS，使用Docker但禁用CGO
            docker run --rm \
                -v $(pwd):/app \
                -w /app \
                -e CGO_ENABLED=0 \
                -e GOOS=darwin \
                -e GOARCH=amd64 \
                golang:1.21 \
                go build -o ./bin/$OUTPUT_NAME ./src/main.go
        else
            echo "Building Windows binary using Docker..."
            # Windows构建
            docker run --rm \
                -v $(pwd):/app \
                -w /app \
                -e CGO_ENABLED=0 \
                -e GOOS=windows \
                -e GOARCH=amd64 \
                golang:1.21 \
                go build -o ./bin/$OUTPUT_NAME ./src/main.go
        fi
    else
        # 如果没有Docker，尝试直接在本地构建
        echo "Docker not available, trying to build locally..."
        if [ "$GOOS" == "linux" ]; then
            # 对于Linux，尝试使用本地Go环境
            if [ "$(uname)" == "Linux" ]; then
                # 如果当前是Linux环境，启用CGO
                echo "Building Linux binary locally with CGO enabled..."
                env CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./bin/$OUTPUT_NAME ./src/main.go
            else
                # 如果不是Linux环境，禁用CGO
                echo "Building Linux binary locally with CGO disabled..."
                env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/$OUTPUT_NAME ./src/main.go
            fi
        elif [ "$GOOS" == "darwin" ]; then
            # 对于macOS，尝试使用本地Go环境
            if [ "$(uname)" == "Darwin" ]; then
                # 如果当前是macOS环境，启用CGO
                echo "Building macOS binary locally with CGO enabled..."
                env CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o ./bin/$OUTPUT_NAME ./src/main.go
            else
                # 如果不是macOS环境，禁用CGO
                echo "Building macOS binary locally with CGO disabled..."
                env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/$OUTPUT_NAME ./src/main.go
            fi
        else
            # Windows构建禁用CGO
            echo "Building Windows binary locally with CGO disabled..."
            env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/$OUTPUT_NAME ./src/main.go
        fi
    fi

    # 检查构建是否成功
    if [ -f "./bin/$OUTPUT_NAME" ]; then
        echo "Successfully built binary for $GOOS/$GOARCH: ./bin/$OUTPUT_NAME"
    else
        echo "Failed to build binary for $GOOS/$GOARCH"
    fi
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
    if command -v docker &> /dev/null; then
        echo "Building Docker image..."
        docker build -t ${APP_NAME}:${VERSION} .
        
        echo "Tagging Docker image..."
        docker tag ${APP_NAME}:${VERSION} ${DOCKER_REPO}/${APP_NAME}:${VERSION}
        docker tag ${APP_NAME}:${VERSION} ${DOCKER_REPO}/${APP_NAME}:latest
        
        echo "Pushing Docker image..."
        docker push ${DOCKER_REPO}/${APP_NAME}:${VERSION}
        docker push ${DOCKER_REPO}/${APP_NAME}:latest
        
        echo "Docker image pushed successfully!"
    else
        echo "Docker is not installed. Cannot build and push Docker image."
    fi
fi