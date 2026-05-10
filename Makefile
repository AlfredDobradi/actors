.PHONY: build run clean

build:
	go build -o bin/game ./examples/game/main.go

run: build
	./bin/game

clean:
	rm -rf bin

lint:
	golangci-lint run --config .golangci.yaml

test:
	go test -v ./...
