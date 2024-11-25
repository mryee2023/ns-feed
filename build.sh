#!/bin/bash

APP_NAME="ns-feed-notifier"
PLATFORMS=("linux/amd64" "windows/amd64" "darwin/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "$PLATFORM"
    OUTPUT_NAME="${APP_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi
    echo "Building for $GOOS/$GOARCH..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -o ./bin/$OUTPUT_NAME ./src/main.go
done