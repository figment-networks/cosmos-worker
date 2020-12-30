package rpc

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cosmos/cosmos-sdk/client/context"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Client is a Tendermint RPC client for cosmos sdk using figmentnetworks datahub
type Client struct {
	cliCtx context.CLIContext
	logger *zap.Logger

	rateLimiter *rate.Limiter
}

// NewClient returns a new client for a given endpoint
func NewClient(nodeURI, key, chainID string, logger *zap.Logger, reqPerSecLimit int) (*Client, error) {
	rateLimiter := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)

	cliCtx, err := makeCliCtx(nodeURI, key, chainID)
	if err != nil {
		fmt.Printf("failted to get client: %v\n", err)
		return nil, err
	}

	cli := &Client{
		logger:      logger,
		cliCtx:      cliCtx,
		rateLimiter: rateLimiter,
	}
	return cli, nil
}

func makeCliCtx(nodeURI, key, chainID string) (context.CLIContext, error) {
	cliCtx := context.NewCLIContext()
	c := &http.Client{
		Timeout:   time.Second * 30,
		Transport: NewDHTransport(key),
	}

	rpc, err := rpchttp.NewWithClient(nodeURI, "/websocket", c)
	if err != nil {
		fmt.Printf("failted to get client: %v\n", err)
		return cliCtx, err
	}

	cdc := makeCodec()
	cliCtx = cliCtx.
		WithCodec(cdc).
		WithClient(rpc).
		WithChainID(chainID)

	return cliCtx, nil
}

func makeCodec() *codec.Codec {
	var cdc = codec.New()
	bank.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)
	distr.RegisterCodec(cdc)
	slashing.RegisterCodec(cdc)
	gov.RegisterCodec(cdc)
	crisis.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	codec.RegisterEvidences(cdc)
	return cdc
}

// // InitMetrics initialise metrics
// func InitMetrics() {
// 	convertionDurationObserver = conversionDuration.WithLabels("conversion")
// 	transactionConversionDuration = conversionDuration.WithLabels("transaction")
// 	blockCacheEfficiencyHit = blockCacheEfficiency.WithLabels("hit")
// 	blockCacheEfficiencyMissed = blockCacheEfficiency.WithLabels("missed")
// 	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
// }
