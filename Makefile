
LDFLAGS      := -w -s
MODULE       := github.com/figment-networks/cosmos-worker
VERSION_FILE ?= ./VERSION


# Git Status
GIT_SHA ?= $(shell git rev-parse --short HEAD)

ifneq (,$(wildcard $(VERSION_FILE)))
VERSION ?= $(shell head -n 1 $(VERSION_FILE))
else
VERSION ?= n/a
endif

all: build

.PHONY: build
build: LDFLAGS += -X $(MODULE)/cmd/worker-cosmos/config.Timestamp=$(shell date +%s)
build: LDFLAGS += -X $(MODULE)/cmd/worker-cosmos/config.Version=$(VERSION)
build: LDFLAGS += -X $(MODULE)/cmd/worker-cosmos/config.GitSHA=$(GIT_SHA)
build:
	go build -o worker -ldflags '$(LDFLAGS)'  ./cmd/worker-cosmos

.PHONY: pack-release
pack-release:
	@mkdir -p ./release
	@make build
	@mv ./worker ./release/worker
	@zip -r cosmos-worker ./release
	@rm -rf ./release
