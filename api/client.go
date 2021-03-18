package api

import (
	"net/http"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

type ClientConfig struct {
	ReqPerSecond        int
	ReqPerSecondLCD     int
	TimeoutBlockCall    time.Duration
	TimeoutSearchTxCall time.Duration
}

// Client
type Client struct {
	logger *zap.Logger
	cli    *grpc.ClientConn
	Sbc    *SimpleBlockCache

	// GRPC
	txServiceClient tx.ServiceClient
	tmServiceClient tmservice.ServiceClient
	rateLimiterGRPC *rate.Limiter

	cfg *ClientConfig

	// LCD
	cosmosLCDAddr  string
	datahubKey     string
	httpClient     *http.Client
	rateLimiterLCD *rate.Limiter
}

// NewClient returns a new client for a given endpoint
func NewClient(logger *zap.Logger, cli *grpc.ClientConn, cfg *ClientConfig, cosmosLCDAddr, datahubKey string) *Client {
	rateLimiterGRPC := rate.NewLimiter(rate.Limit(cfg.ReqPerSecond), cfg.ReqPerSecond)
	rateLimiterLCD := rate.NewLimiter(rate.Limit(cfg.ReqPerSecondLCD), cfg.ReqPerSecondLCD)

	return &Client{
		logger:          logger,
		Sbc:             NewSimpleBlockCache(400),
		tmServiceClient: tmservice.NewServiceClient(cli),
		txServiceClient: tx.NewServiceClient(cli),
		rateLimiterGRPC: rateLimiterGRPC,
		cosmosLCDAddr:   cosmosLCDAddr,
		datahubKey:      datahubKey,
		cfg:             cfg,
		rateLimiterLCD:  rateLimiterLCD,
		httpClient: &http.Client{
			Timeout: time.Second * 40,
		},
	}
}

// InitMetrics initialise metrics
func InitMetrics() {
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
}
