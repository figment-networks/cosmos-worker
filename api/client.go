package api

import (
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

// Client  ads
type Client struct {
	logger *zap.Logger
	cli    *grpc.ClientConn
	Sbc    *SimpleBlockCache

	txServiceClient tx.ServiceClient

	tmServiceClient tmservice.ServiceClient
	rateLimiter     *rate.Limiter
}

// NewClient returns a new client for a given endpoint
func NewClient(logger *zap.Logger, cli *grpc.ClientConn, reqPerSecLimit int) *Client {
	/*	if c == nil {
			c = &http.Client{
				Timeout: time.Second * 40,
			}
		}
	*/

	/*
	   	rateLimiter := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)

	    	grpcRes, err := s.queryClient.GetTxsEvent(
	   		context.Background(),
	   		&tx.GetTxsEventRequest{Event: "message.action=send",
	   			Pagination: &query.PageRequest{
	   				CountTotal: false,
	   				Offset:     0,
	   				Limit:      1,
	   			},
	   		},
	   	)
	*/

	rateLimiter := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)

	return &Client{
		logger:          logger,
		rateLimiter:     rateLimiter,
		Sbc:             NewSimpleBlockCache(400),
		tmServiceClient: tmservice.NewServiceClient(cli),
		txServiceClient: tx.NewServiceClient(cli),
	}
}

// InitMetrics initialise metrics
func InitMetrics() {
	convertionDurationObserver = conversionDuration.WithLabels("conversion")
	transactionConversionDuration = conversionDuration.WithLabels("transaction")
	blockCacheEfficiencyHit = blockCacheEfficiency.WithLabels("hit")
	blockCacheEfficiencyMissed = blockCacheEfficiency.WithLabels("missed")
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
}
