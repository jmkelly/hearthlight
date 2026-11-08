BINARY := hearthlight
VERSION ?= 0.1.0
GO ?= go

.PHONY: build run demo test vet fmt fmt-check check clean

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

check: fmt-check vet test

clean:
	rm -rf bin
