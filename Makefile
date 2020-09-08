SRC := $(wildcard *.go)

.ONESHELL:
.PHONY: build build-release test test-go test-regression setup-tests

build: $(SRC)
	go build -v

build-release:
	# adapted from [[https://github.com/gokcehan/lf/blob/1d959910547f6b69e0898373063e1b3e7525a4ec/gen/xbuild.sh][gokcehan/lf]].
	[ -z $$version ] && version=$$(git describe --tags)

	mkdir -p dist
	build()     { go build -ldflags="-s -w -X main.gVersion=$$version" -o dotty     && tar czf dist/dotty-$${GOOS}-$${GOARCH}.tar.gz   dotty       --remove-files; }
	build_win() { go build -ldflags="-s -w -X main.gVersion=$$version" -o dotty.exe && zip     dist/dotty-$${GOOS}-$${GOARCH}.zip      dotty.exe   --move; }

	CGO_ENABLED=0 GOOS=darwin    GOARCH=amd64    build
	CGO_ENABLED=0 GOOS=dragonfly GOARCH=amd64    build
	CGO_ENABLED=0 GOOS=freebsd   GOARCH=386      build
	CGO_ENABLED=0 GOOS=freebsd   GOARCH=amd64    build
	CGO_ENABLED=0 GOOS=freebsd   GOARCH=arm      build
	CGO_ENABLED=0 GOOS=linux     GOARCH=386      build
	CGO_ENABLED=0 GOOS=linux     GOARCH=amd64    build
	CGO_ENABLED=0 GOOS=linux     GOARCH=arm      build
	CGO_ENABLED=0 GOOS=linux     GOARCH=arm64    build
	CGO_ENABLED=0 GOOS=linux     GOARCH=ppc64    build
	CGO_ENABLED=0 GOOS=linux     GOARCH=ppc64le  build
	CGO_ENABLED=0 GOOS=linux     GOARCH=mips     build
	CGO_ENABLED=0 GOOS=linux     GOARCH=mipsle   build
	CGO_ENABLED=0 GOOS=linux     GOARCH=mips64   build
	CGO_ENABLED=0 GOOS=linux     GOARCH=mips64le build
	CGO_ENABLED=0 GOOS=netbsd    GOARCH=386      build
	CGO_ENABLED=0 GOOS=netbsd    GOARCH=amd64    build
	CGO_ENABLED=0 GOOS=netbsd    GOARCH=arm      build
	CGO_ENABLED=0 GOOS=openbsd   GOARCH=386      build
	CGO_ENABLED=0 GOOS=openbsd   GOARCH=amd64    build
	CGO_ENABLED=0 GOOS=openbsd   GOARCH=arm      build

	CGO_ENABLED=0 GOOS=windows   GOARCH=386      build_win
	CGO_ENABLED=0 GOOS=windows   GOARCH=amd64    build_win

test: test-go test-regression

test-go:
	go test

setup-tests:
	cd ./tests/
	bundle i

test-regression: build
	cd ./tests/
	bundle exec rspec --profile -P 'test_*.rb'

clean:
	go clean
