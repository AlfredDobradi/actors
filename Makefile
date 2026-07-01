.PHONY: build run clean

VERBOSE_FLAG := $(if $(VERBOSE),-v,)

build:
	go build -o bin/game ./cmd/game/main.go

run: build
	./bin/game

clean:
	rm -rf bin

lint:
	golangci-lint run --config .golangci.yaml

test:
	go test $(VERBOSE_FLAG) ./...