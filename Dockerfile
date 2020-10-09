# ------------------------------------------------------------------------------
# Builder Image
# ------------------------------------------------------------------------------
FROM golang:1.14 AS build

WORKDIR /go/src/github.com/figment-networks/cosmos-worker/

COPY ./go.mod .
COPY ./go.sum .

RUN go mod download

COPY .git .git
COPY ./Makefile ./Makefile
COPY ./api ./api
COPY ./client ./client
COPY ./cmd/common ./cmd/common
COPY ./cmd/worker-cosmos ./cmd/worker-cosmos

ENV CGO_ENABLED=0
ENV GOARCH=amd64
ENV GOOS=linux

RUN \
  GO_VERSION=$(go version | awk {'print $3'}) \
  GIT_COMMIT=$(git rev-parse HEAD) \
  make build

# ------------------------------------------------------------------------------
# Target Image
# ------------------------------------------------------------------------------
FROM alpine:3.10 AS release

WORKDIR /app/cosmos
COPY --from=build /go/src/github.com/figment-networks/cosmos-worker/cosmos-worker /app/cosmos/worker
RUN chmod a+x ./worker
CMD ["./worker"]
