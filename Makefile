.PHONY: build clean install tidy test universal

BIN      := wechattweak
PKG      := ./cmd/wechattweak
PREFIX   ?= /usr/local

# 单架构本地构建
build:
	go build -trimpath -ldflags "-s -w" -o $(BIN) $(PKG)

# 同时构建 arm64 + amd64，并合并为 universal binary（macOS only，需要 lipo）
universal:
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o $(BIN)-arm64 $(PKG)
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(BIN)-amd64 $(PKG)
	lipo -create -output $(BIN) $(BIN)-arm64 $(BIN)-amd64
	rm -f $(BIN)-arm64 $(BIN)-amd64

install: universal
	install -m 0755 $(BIN) $(PREFIX)/bin/$(BIN)

tidy:
	go mod tidy

test:
	go test ./...

clean:
	rm -f $(BIN) $(BIN)-arm64 $(BIN)-amd64
