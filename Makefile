APP_NAME := quicknotes
DATA_FILE ?= ./data/notes.json

.PHONY: run test build fmt clean

run:
	QUICKNOTES_DATA_FILE=$(DATA_FILE) go run ./cmd/server

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/server

fmt:
	gofmt -w ./cmd ./internal

clean:
	rm -rf ./bin
