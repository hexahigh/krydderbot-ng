GIT_COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_VERSION=$(shell go version | cut -d " " -f 3)

build:
	go build -ldflags "-X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)" -o krydderbot-ng

release: build
	strip -s krydderbot-ng
	upx krydderbot-ng
