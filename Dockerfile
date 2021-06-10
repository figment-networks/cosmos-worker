# ------------------------------------------------------------------------------
# Builder Image
# ------------------------------------------------------------------------------
FROM golang AS build

WORKDIR /build

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

RUN make build

# ------------------------------------------------------------------------------
# Target Image
# ------------------------------------------------------------------------------
FROM alpine AS release

WORKDIR /app
COPY --from=build /build/worker /app/worker

RUN addgroup --gid 1234 figment
RUN adduser --system --uid 1234 figment

RUN chown -R figment:figment /app/worker

USER 1234

CMD ["/app/worker"]
