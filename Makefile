# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=ns-feed-bot
MAIN_PATH=./src/main.go

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_DIR=build

all: clean build

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete. Binary location: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)

test:
	$(GOTEST) -v ./...

deps:
	$(GOGET) -v -d ./...
	$(GOMOD) tidy

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: all build clean test deps run
