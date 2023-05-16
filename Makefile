#flags=-ldflags="-s -w"
flags=-ldflags="-s -w -extldflags -static"
TAG := $(shell git tag | sed -e "s,v,,g" | sort -r | head -n 1)

all: build

vet:
	go vet .

build:
	go clean; rm -rf pkg; CGO_ENABLED=0 go build -o mlhub ${flags}

build_debug:
	go clean; rm -rf pkg; CGO_ENABLED=0 go build -o mlhub ${flags} -gcflags="-m -m"

build_amd64: build_linux

build_darwin:
	go clean; rm -rf pkg mlhub; GOOS=darwin CGO_ENABLED=0 go build -o mlhub ${flags}

build_linux:
	go clean; rm -rf pkg mlhub; GOOS=linux CGO_ENABLED=0 go build -o mlhub ${flags}
	mkdir -p /tmp/mlhub/amd64
	cp mlhub /tmp/mlhub/amd64

build_power8:
	go clean; rm -rf pkg mlhub; GOARCH=ppc64le GOOS=linux CGO_ENABLED=0 go build -o mlhub ${flags}
	mkdir -p /tmp/mlhub/power8
	cp mlhub /tmp/mlhub/power8

build_arm64:
	go clean; rm -rf pkg mlhub; GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o mlhub ${flags}
	mkdir -p /tmp/mlhub/arm64
	cp mlhub /tmp/mlhub/arm64

build_windows:
	go clean; rm -rf pkg mlhub; GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -o mlhub ${flags}
	mkdir -p /tmp/mlhub/windows
	cp mlhub /tmp/mlhub/windows

install:
	go install

clean:
	go clean; rm -rf pkg; rm -rf mlhub

test : test1

test1:
	go test -v -bench=.

tarball:
	cp -r /tmp/mlhub .
	tar cfz mlhub.tar.gz mlhub
	rm -rf /tmp/mlhub

release: clean build_amd64 build_arm64 build_windows build_power8 build_darwin tarball
