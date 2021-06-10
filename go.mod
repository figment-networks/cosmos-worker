module github.com/figment-networks/cosmos-worker

go 1.16

require (
	github.com/bearcherian/rollzap v1.0.2
	github.com/cosmos/cosmos-sdk v0.42.5
	github.com/figment-networks/indexer-manager v0.4.0-rc1.0.20210610142546-3a018ba4093b
	github.com/figment-networks/indexing-engine v0.3.2-0.20210603103553-9df604641a66
	github.com/gogo/protobuf v1.3.3
	github.com/google/uuid v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/rollbar/rollbar-go v1.4.0
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.10
	go.uber.org/zap v1.17.0
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4
	google.golang.org/grpc v1.37.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.33.2

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
