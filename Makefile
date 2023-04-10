VERSION=`git rev-parse --short HEAD`
#flags=-ldflags="-s -w -X main.version=${VERSION}"
OS := $(shell uname)
ifeq ($(OS),Darwin)
flags=-ldflags="-s -w -X main.version=${VERSION}"
else
flags=-ldflags="-s -w -X main.version=${VERSION} -extldflags -static"
endif

all: build

vet:
	go vet .

build:
	go clean; rm -rf pkg; CGO_ENABLED=0 go build -o mlaas-proxy ${flags}

build_debug:
	go clean; rm -rf pkg; CGO_ENABLED=0 go build -o mlaas-proxy ${flags} -gcflags="-m -m"

build_amd64: build_linux

build_darwin:
	go clean; rm -rf pkg mlaas-proxy; GOOS=darwin CGO_ENABLED=0 go build -o mlaas-proxy ${flags}

build_linux:
	go clean; rm -rf pkg mlaas-proxy; GOOS=linux CGO_ENABLED=0 go build -o mlaas-proxy ${flags}
	mkdir -p /tmp/auth-proxy-tools/amd64
	cp mlaas-proxy /tmp/auth-proxy-tools/amd64

build_power8:
	go clean; rm -rf pkg mlaas-proxy; GOARCH=ppc64le GOOS=linux CGO_ENABLED=0 go build -o mlaas-proxy ${flags}
	mkdir -p /tmp/auth-proxy-tools/power8
	cp mlaas-proxy /tmp/auth-proxy-tools/power8

build_arm64:
	go clean; rm -rf pkg mlaas-proxy; GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o mlaas-proxy ${flags}
	mkdir -p /tmp/auth-proxy-tools/arm64
	cp mlaas-proxy /tmp/auth-proxy-tools/arm64

build_windows:
	go clean; rm -rf pkg mlaas-proxy; GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -o mlaas-proxy ${flags}
	mkdir -p /tmp/auth-proxy-tools/windows
	cp mlaas-proxy /tmp/auth-proxy-tools/windows

install:
	go install

clean:
	go clean; rm -rf pkg; rm -rf auth-proxy-tools

test : test1

test1:
	go test -v -bench=.

tarball:
	cp -r /tmp/auth-proxy-tools .
	tar cfz auth-proxy-tools.tar.gz auth-proxy-tools
	rm -rf /tmp/auth-proxy-tools

release: clean build_amd64 build_arm64 build_windows build_power8 build_darwin tarball
