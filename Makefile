all: build

.PHONY: build
build:
	go build -o worker ./cmd/worker-cosmos

.PHONY: pack-release
pack-release:
	@mkdir -p ./release
	@make build
	@zip -r cosmos-worker ./release
	@rm -rf ./release
