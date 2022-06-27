GO = go
PYTHON = python3
SCDOC = scdoc
TAG_COMMIT = $(shell git rev-list --tags --max-count=1)
TAG_VERSION = $(shell git describe --tags ${TAG_COMMIT})
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = "-s -w -X main.version=$(TAG_VERSION) -X main.build=$(BUILD_DATE)"

pkgs = $(shell $(GO) list ./... | grep -v /vendor/)

.PHONY: all clean upgrade docs

all:
	$(GO) build

fmt:
	$(GO) fmt

clean:
	rm -f shoelaces docs/shoelaces.8

deps:
	$(GO) build -v $(EXTRA_FLAGS) ./...
	$(PYTHON) -m pip install --user pytest python-dateutil

upgrade: deps
	go get -u ./...
	go mod tidy
	go mod vendor

shoelaces.8:
	$(SCDOC) < docs/shoelaces.8.scd > docs/shoelaces.8

docs: shoelaces.8

test: fmt
	$(GO) test -v $(pkgs) && \
	./test/integ-test/integ_test.py

binaries: deps linux windows macos

linux:
	GOOS=linux ${GO} build -o bin/shoelaces -ldflags ${LDFLAGS}

windows:
	GOOS=windows ${GO} build -o bin/shoelaces.exe -ldflags ${LDFLAGS}

macos:
	GOOS=darwin ${GO} build -o bin/shoelaces -ldflags ${LDFLAGS}
