.PHONY: all build clean test

# 项目名称
PROJECT_NAME := free-agent
# 输出目录
OUTPUT_DIR := bin
# 主入口
MAIN_PATH := cmd/free-agent/main.go

# 默认目标：编译并输出到 bin 目录
all: build

# 编译主程序
build:
	@echo "Building $(PROJECT_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(PROJECT_NAME).exe $(MAIN_PATH)
	@echo "Build completed: $(OUTPUT_DIR)/$(PROJECT_NAME).exe"

# 清理构建产物
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(OUTPUT_DIR)/*.exe
	rm -f *.exe
	@echo "Clean completed"

# 运行测试
test:
	@echo "Running tests..."
	go test ./internal/...

# 编译并运行（开发模式）
run: build
	@echo "Running $(PROJECT_NAME)..."
	./$(OUTPUT_DIR)/$(PROJECT_NAME).exe

# 编译所有子命令
build-all: build
	@echo "Building additional tools..."
	go build -o $(OUTPUT_DIR)/vds-test.exe ./cmd/vds-test/main.go
	@echo "All builds completed"
