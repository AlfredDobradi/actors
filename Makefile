.PHONY: build run clean

LOG_LEVEL ?= info
LOG_FORMAT ?= json

build:
	go build -o bin/game ./examples/game/main.go

run: build
	LOG_LEVEL=$(LOG_LEVEL) LOG_FORMAT=$(LOG_FORMAT) ./bin/game

clean:
	rm -rf bin

lint:
	golangci-lint run --config .golangci.yaml
