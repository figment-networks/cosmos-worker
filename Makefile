all: build

.PHONY: build
build:
	go build -o worker-cosmos ./cmd/worker-cosmos

