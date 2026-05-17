.PHONY: build run dev test lint clean install release setup

BINARY  := ytcap
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

setup:
	mise install

build:
	go build $(LDFLAGS) -o $(BINARY) .

run:
	go run . $(ARGS)

dev:
	YTCAP_LOG=debug go run . $(ARGS)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

install:
	go install $(LDFLAGS) .

release:
	goreleaser release --clean
