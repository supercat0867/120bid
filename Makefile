# Makefile

APP_NAME=120bid

# 默认构建（当前系统）
build:
	go build -o build/$(APP_NAME) .

# 构建 Windows 版本
build-win:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o build/$(APP_NAME).exe .

# 构建 macOS 版本
build-mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/$(APP_NAME)_mac .

# 构建 Linux 版本
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/$(APP_NAME)_linux .

# 清理输出
clean:
	rm -rf build