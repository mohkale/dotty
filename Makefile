CC := go

SRC := $(wildcard pkg/*.go)
BIN := bin
CMD := dotty

.ONESHELL:
.PHONY: build build-release test test-go test-regression setup-tests list

define create-build-cmd
$(BIN)/$1: $(SRC) $(wildcard cmd/$1/*.go)
	echo CC dotty
	mkdir -p "$(BIN)"
	$(CC) build -v -o $(BIN)/$1 ./cmd/$1
endef
$(foreach binary,$(CMD),$(eval $(call create-build-cmd,$(binary))))

build: $(BIN)/dotty

# Adapted from [[https://github.com/gokcehan/lf/blob/1d959910547f6b69e0898373063e1b3e7525a4ec/gen/xbuild.sh][gokcehan/lf]].
build-release:
	[ -z $$version ] && version=$$(git tag | tail -n1)

	mkdir -p $(BIN)/release
	build()     { $(CC) build -ldflags="-s -w -X main.gVersion=$$version" -o $(BIN)/release/dotty     $${1} && tar czf $(BIN)/release/dotty-$${GOOS}-$${GOARCH}.tar.gz   $(BIN)/release/dotty       --remove-files; }
	build_win() { $(CC) build -ldflags="-s -w -X main.gVersion=$$version" -o $(BIN)/release/dotty.exe $${1} && tar czf $(BIN)/release/dotty-$${GOOS}-$${GOARCH}.tar.gz   $(BIN)/release/dotty.exe   --remove-files; }

	CGO_ENABLED=0 GOOS=darwin    GOARCH=amd64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=dragonfly GOARCH=amd64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=freebsd   GOARCH=386      build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=freebsd   GOARCH=amd64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=freebsd   GOARCH=arm      build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=386      build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=amd64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=arm      build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=arm64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=ppc64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=ppc64le  build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=mips     build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=mipsle   build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=mips64   build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=linux     GOARCH=mips64le build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=netbsd    GOARCH=386      build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=netbsd    GOARCH=amd64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=netbsd    GOARCH=arm      build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=openbsd   GOARCH=386      build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=openbsd   GOARCH=amd64    build ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=openbsd   GOARCH=arm      build ./cmd/$(CMD)

	CGO_ENABLED=0 GOOS=windows   GOARCH=386      build_win ./cmd/$(CMD)
	CGO_ENABLED=0 GOOS=windows   GOARCH=amd64    build_win ./cmd/$(CMD)

test: test-go test-regression

test-go:
	@echo TEST Go
	go test ./pkg/

setup-tests:
	@echo TEST Setup Ruby
	cd ./tests/
	bundle i

test-regression: build
	@echo TEST Ruby
	cd ./tests/
	bundle exec rspec --profile -P 'test_*.rb'

clean:
	go clean

$(V).SILENT: # Assign the V environment variable to not be silent
