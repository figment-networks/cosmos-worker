package api

import (
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"net/http"
	"time"
)

// Client  ads
type Client struct {
	logger *zap.Logger
	cli    *grpc.ClientConn
	Sbc    *SimpleBlockCache

	// GRPC
	txServiceClient tx.ServiceClient
	tmServiceClient tmservice.ServiceClient
	rateLimiterGRPC *rate.Limiter

	// LCD
	cosmosLCDAddr  string
	datahubKey     string
	httpClient     *http.Client
	rateLimiterLCD *rate.Limiter
}

// NewClient returns a new client for a given endpoint
func NewClient(logger *zap.Logger, cli *grpc.ClientConn, reqPerSecLimit int, cosmosLCDAddr, datahubKey string) *Client {
	rateLimiterGRPC := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)
	rateLimiterLCD := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)

	return &Client{
		logger:          logger,
		Sbc:             NewSimpleBlockCache(400),
		tmServiceClient: tmservice.NewServiceClient(cli),
		txServiceClient: tx.NewServiceClient(cli),
		rateLimiterGRPC: rateLimiterGRPC,
		cosmosLCDAddr:   cosmosLCDAddr,
		datahubKey:      datahubKey,
		rateLimiterLCD:  rateLimiterLCD,
		httpClient: &http.Client{
			Timeout: time.Second * 40,
		},
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
