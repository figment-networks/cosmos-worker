package client

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/figment-networks/indexing-engine/metrics"

	"github.com/figment-networks/indexer-manager/structs"

	"github.com/figment-networks/indexer-manager/worker/store"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/figment-networks/cosmos-worker/api"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
)

const page = 100
const blockchainEndpointLimit = 20

var (
	getTransactionDuration        *metrics.GroupObserver
	getLatestDuration             *metrics.GroupObserver
	getBlockDuration              *metrics.GroupObserver
	getRewardDuration             *metrics.GroupObserver
	getAccountBalanceDuration     *metrics.GroupObserver
	getAccountDelegationsDuration *metrics.GroupObserver
)

type GRPC interface {
	GetBlock(ctx context.Context, params structs.HeightHash) (block structs.Block, er error)
	SearchTx(ctx context.Context, r structs.HeightHash, block structs.Block, perPage uint64) (txs []structs.Transaction, err error)
	GetReward(ctx context.Context, params structs.HeightAccount) (resp structs.GetRewardResponse, err error)
	GetAccountBalance(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountBalanceResponse, err error)
	GetAccountDelegations(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountDelegationsResponse, err error)
}

type OutputSender interface {
	Send(cStructs.TaskResponse) error
}

// IndexerClient is implementation of a client (main worker code)
type IndexerClient struct {
	grpc GRPC

	logger  *zap.Logger
	streams map[uuid.UUID]*cStructs.StreamAccess
	sLock   sync.Mutex

	storeClient store.HeightStore

	maximumHeightsToGet uint64
}

// NewIndexerClient is IndexerClient constructor
func NewIndexerClient(ctx context.Context, logger *zap.Logger, grpc GRPC, storeClient store.HeightStore, maximumHeightsToGet uint64) *IndexerClient {
	getTransactionDuration = endpointDuration.WithLabels("getTransactions")
	getLatestDuration = endpointDuration.WithLabels("getLatest")
	getBlockDuration = endpointDuration.WithLabels("getBlock")
	getRewardDuration = endpointDuration.WithLabels("getReward")
	getAccountBalanceDuration = endpointDuration.WithLabels("getAccountBalance")
	getAccountDelegationsDuration = endpointDuration.WithLabels("getAccountDelegations")
	api.InitMetrics()

	return &IndexerClient{
		logger:              logger,
		grpc:                grpc,
		maximumHeightsToGet: maximumHeightsToGet,
		storeClient:         storeClient,
		streams:             make(map[uuid.UUID]*cStructs.StreamAccess),
	}
}

// CloseStream removes stream from worker/client
func (ic *IndexerClient) CloseStream(ctx context.Context, streamID uuid.UUID) error {
	ic.sLock.Lock()
	defer ic.sLock.Unlock()

	ic.logger.Debug("[COSMOS-CLIENT] Close Stream", zap.Stringer("streamID", streamID))
	delete(ic.streams, streamID)

	return nil
}

// RegisterStream adds new listeners to the streams - currently fixed number per stream
func (ic *IndexerClient) RegisterStream(ctx context.Context, stream *cStructs.StreamAccess) error {
	ic.logger.Debug("[COSMOS-CLIENT] Register Stream", zap.Stringer("streamID", stream.StreamID))
	newStreamsMetric.WithLabels().Inc()

	ic.sLock.Lock()
	defer ic.sLock.Unlock()
	ic.streams[stream.StreamID] = stream

	// Limit workers not to create new goroutines over and over again
	for i := 0; i < 20; i++ {
		go ic.Run(ctx, stream)
	}

	return nil
}

// Run listens on the stream events (new tasks)
func (ic *IndexerClient) Run(ctx context.Context, stream *cStructs.StreamAccess) {
	for {
		select {
		case <-ctx.Done():
			ic.sLock.Lock()
			delete(ic.streams, stream.StreamID)
			ic.sLock.Unlock()
			return
		case <-stream.Finish:
			return
		case taskRequest := <-stream.RequestListener:
			receivedRequestsMetric.WithLabels(taskRequest.Type).Inc()
			tctx, cancel := context.WithTimeout(ctx, time.Minute*10)
			switch taskRequest.Type {
			case structs.ReqIDGetTransactions:
				ic.GetTransactions(tctx, taskRequest, stream, ic.grpc)
			case structs.ReqIDGetReward:
				ic.GetReward(tctx, taskRequest, stream, ic.grpc)
			case structs.ReqIDAccountBalance:
				ic.GetAccountBalance(tctx, taskRequest, stream, ic.grpc)
			case structs.ReqIDAccountDelegations:
				ic.GetAccountDelegations(tctx, taskRequest, stream, ic.grpc)
			case structs.ReqIDGetLatestMark:
				ic.GetLatestMark(tctx, taskRequest, stream, ic.grpc)
			default:
				stream.Send(cStructs.TaskResponse{
					Id:    taskRequest.Id,
					Error: cStructs.TaskError{Msg: "There is no such handler " + taskRequest.Type},
					Final: true,
				})
			}
			cancel()
		}
	}
}

// GetBlock gets block
func (ic *IndexerClient) GetBlock(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client GRPC) {
	timer := metrics.NewTimer(getBlockDuration)
	defer timer.ObserveDuration()

	hr := &structs.HeightHash{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"},
			Final: true,
		})
		return
	}

	block, err := client.GetBlock(ctx, *hr)
	if err != nil {
		ic.logger.Error("Error getting block", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting block data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "Block",
		Payload: block,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

// GetAccountBalance gets account balance
func (ic *IndexerClient) GetAccountBalance(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client GRPC) {
	timer := metrics.NewTimer(getAccountBalanceDuration)
	defer timer.ObserveDuration()

	ha := &structs.HeightAccount{}
	err := json.Unmarshal(tr.Payload, ha)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	blnc, err := client.GetAccountBalance(sCtx, *ha)
	if err != nil {
		ic.logger.Error("Error getting account balance", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting account balance data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "AccountBalance",
		Payload: blnc,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

// GetAccountDelegations gets account delegations
func (ic *IndexerClient) GetAccountDelegations(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client GRPC) {
	timer := metrics.NewTimer(getAccountDelegationsDuration)
	defer timer.ObserveDuration()

	ha := &structs.HeightAccount{}
	err := json.Unmarshal(tr.Payload, ha)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	blnc, err := client.GetAccountDelegations(sCtx, *ha)
	if err != nil {
		ic.logger.Error("Error getting account balance", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting account balance data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "AccountDelegations",
		Payload: blnc,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

// GetReward gets reward
func (ic *IndexerClient) GetReward(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client GRPC) {
	timer := metrics.NewTimer(getRewardDuration)
	defer timer.ObserveDuration()

	ha := &structs.HeightAccount{}
	err := json.Unmarshal(tr.Payload, ha)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	reward, err := client.GetReward(sCtx, *ha)
	if err != nil {
		ic.logger.Error("Error getting reward", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting reward data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "Reward",
		Payload: reward,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

// sendResp constructs protocol response and send it out to transport
func sendResp(ctx context.Context, id uuid.UUID, in <-chan cStructs.OutResp, logger *zap.Logger, sender OutputSender, fin chan bool) {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	order := uint64(0)

	var contextDone bool

SendLoop:
	for {
		select {
		case <-ctx.Done():
			contextDone = true
			break SendLoop
		case t, ok := <-in:
			if !ok && t.Type == "" {
				break SendLoop
			}
			b.Reset()

			err := enc.Encode(t.Payload)
			if err != nil {
				logger.Error("[COSMOS-CLIENT] Error encoding payload data", zap.Error(err))
			}

			tr := cStructs.TaskResponse{
				Id:      id,
				Type:    t.Type,
				Order:   order,
				Payload: make([]byte, b.Len()),
			}

			b.Read(tr.Payload)
			order++
			err = sender.Send(tr)
			if err != nil {
				logger.Error("[COSMOS-CLIENT] Error sending data", zap.Error(err))
			}
			sendResponseMetric.WithLabels(t.Type, "yes").Inc()
		}
	}

	err := sender.Send(cStructs.TaskResponse{
		Id:    id,
		Type:  "END",
		Order: order,
		Final: true,
	})

	if err != nil {
		logger.Error("[COSMOS-CLIENT] Error sending end", zap.Error(err))
	}

	if fin != nil {
		if !contextDone {
			fin <- true
		}
		close(fin)
	}

}
