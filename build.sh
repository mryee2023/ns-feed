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

# 定义支持的平台
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm/v7"
    "linux/arm/v6"
    "linux/386"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
)

# 构建各平台版本
for platform in "${PLATFORMS[@]}"; do
    # 解析平台信息
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    
    # 处理特殊情况：arm 架构的版本号
    if [[ $GOARCH == arm/* ]]; then
        GOARM=${GOARCH#arm/v}
        GOARCH="arm"
    else
        GOARM=""
    fi
    
    # 构建输出文件名
    OUTPUT_NAME="${APP_NAME}-${GOOS}"
    if [ "$GOARCH" == "arm" ]; then
        OUTPUT_NAME+="-armv${GOARM}"
    else
        OUTPUT_NAME+="-${GOARCH}"
    fi
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi
    
    echo "Building for $GOOS/$GOARCH${GOARM:+v$GOARM}..."
    
    # 设置构建环境变量
    BUILD_ENV=(
        "CGO_ENABLED=0"
        "GOOS=$GOOS"
        "GOARCH=$GOARCH"
    )
    if [ -n "$GOARM" ]; then
        BUILD_ENV+=("GOARM=$GOARM")
    fi
    
    # 执行构建
    docker run --rm \
        -v $(pwd):/go/src/app \
        -w /go/src/app \
        $(printf -- "-e %s " "${BUILD_ENV[@]}") \
        golang:1.22 \
        go build -o ./bin/$OUTPUT_NAME ./src/main.go

    # 生成 SHA256 校验和文件
    echo "Generating SHA256 checksum for $OUTPUT_NAME..."
    if command -v sha256sum >/dev/null 2>&1; then
        (cd bin && sha256sum $OUTPUT_NAME > ${OUTPUT_NAME}.sha256)
    else
        # macOS 使用 shasum 命令
        (cd bin && shasum -a 256 $OUTPUT_NAME > ${OUTPUT_NAME}.sha256)
    fi
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