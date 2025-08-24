# BaseSQL Makefile

.PHONY: build install clean test example cli help

# 默认目标
all: build

# 构建 CLI 工具
build:
	@echo "构建 BaseSQL CLI..."
	go build -o bin/basesql ./cmd/basesql

# 安装到系统路径
install: build
	@echo "安装 BaseSQL CLI 到系统路径..."
	sudo cp bin/basesql /usr/local/bin/
	@echo "✅ 安装完成！现在可以在任何地方使用 'basesql' 命令"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/

# 运行测试
test:
	@echo "运行测试..."
	go test ./...

# 运行示例
example:
	@echo "运行示例程序..."
	cd example && go run main.go

# 构建并运行 CLI
cli: build
	@echo "启动 BaseSQL CLI..."
	./bin/basesql

# 显示帮助
help:
	@echo "BaseSQL 构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build    - 构建 CLI 工具"
	@echo "  install  - 安装到系统路径"
	@echo "  clean    - 清理构建文件"
	@echo "  test     - 运行测试"
	@echo "  example  - 运行示例程序"
	@echo "  cli      - 构建并运行 CLI"
	@echo "  help     - 显示此帮助信息"