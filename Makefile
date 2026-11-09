BINARY := hearthlight
VERSION ?= $(shell cat VERSION 2>/dev/null || echo 0.0.0-dev)
GO ?= go

.PHONY: build run demo test vet fmt fmt-check check clean themes

build:
	$(GO) build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/$(BINARY) .

run:
	$(GO) run .

demo:
	$(GO) run . -demo

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

themes:
	{ printf 'window.HearthlightThemes = '; cat web/themes.json; printf ';\n'; } > web/themes.js

check: fmt-check vet test

clean:
	rm -rf bin
