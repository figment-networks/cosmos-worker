module github.com/figment-networks/cosmos-worker

go 1.15

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require (
	github.com/bearcherian/rollzap v1.0.2
	github.com/cosmos/cosmos-sdk v0.40.0
	github.com/figment-networks/indexer-manager v0.1.0
	github.com/figment-networks/indexing-engine v0.1.14
	github.com/gogo/protobuf v1.3.1
	github.com/google/uuid v1.1.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/rollbar/rollbar-go v1.2.0
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/tendermint v0.34.1
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324
	google.golang.org/grpc v1.34.0
)
