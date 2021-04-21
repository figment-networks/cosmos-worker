package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/figment-networks/indexer-manager/structs"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/indexer-manager/worker/store"
	"github.com/figment-networks/indexing-engine/metrics"
	"go.uber.org/zap"
)

// GetTransactions gets new transactions and blocks from cosmos for given range
func (ic *IndexerClient) GetTransactions(ctx context.Context, tr cStructs.TaskRequest, stream OutputSender, client GRPC) {
	timer := metrics.NewTimer(getTransactionDuration)
	defer timer.ObserveDuration()

	hr := &structs.HeightRange{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		ic.logger.Debug("[COSMOS-CLIENT] Cannot unmarshal payload", zap.String("contents", string(tr.Payload)))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "cannot unmarshal payload: " + err.Error()},
			Final: true,
		})
		return
	}
	if hr.EndHeight == 0 {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "end height is zero" + err.Error()},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ic.logger.Debug("[COSMOS-CLIENT] Getting Range", zap.Stringer("taskID", tr.Id), zap.Uint64("start", hr.StartHeight), zap.Uint64("end", hr.EndHeight))
	heights, err := getRange(sCtx, ic.logger, client, *hr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: err.Error()},
			Final: true,
		})
		ic.logger.Error("[COSMOS-CLIENT] Error getting range (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
		return
	}

	err = stream.Send(cStructs.TaskResponse{
		Id:    tr.Id,
		Type:  "Heights",
		Order: 0,
		//Payload: ,
		Final: true,
	})

	ic.logger.Debug("[COSMOS-CLIENT] Finished sending all", zap.Stringer("taskID", tr.Id), zap.Any("heights", hr))

}

func blockAndTx(ctx context.Context, logger *zap.Logger, client GRPC, height uint64) (blockWM structs.BlockWithMeta, txsWM []structs.TransactionWithMeta, err error) {
	defer logger.Sync()
	logger.Debug("[COSMOS-CLIENT] Getting block", zap.Uint64("block", height))

	blockWM = structs.BlockWithMeta{
		Network: "cosmos",
		ChainID: "cosmoshub-4",
		Version: "0.0.1",
	}
	blockWM.Block, err = client.GetBlock(ctx, structs.HeightHash{Height: uint64(height)})

	if err != nil {
		logger.Debug("[COSMOS-CLIENT] Err Getting block", zap.Uint64("block", height), zap.Error(err), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
		return blockWM, nil, fmt.Errorf("error fetching block: %d %w ", uint64(height), err)
	}

	if blockWM.Block.NumberOfTransactions > 0 {
		logger.Debug("[COSMOS-CLIENT] Getting txs", zap.Uint64("block", height), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
		var txs []structs.Transaction
		txs, err = client.SearchTx(ctx, structs.HeightHash{Height: height}, blockWM.Block, page)
		for _, t := range txs {
			txsWM = append(txsWM, structs.TransactionWithMeta{Network: "cosmos", ChainID: "cosmoshub-4", Version: "0.0.1", Transaction: t})
		}
		logger.Debug("[COSMOS-CLIENT] txErr Getting txs", zap.Uint64("block", height), zap.Error(err), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
	}

	logger.Debug("[COSMOS-CLIENT] Got block", zap.Uint64("block", height), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
	return blockWM, txsWM, err
}

func asyncBlockAndTx(ctx context.Context, logger *zap.Logger, wg *sync.WaitGroup, hstore store.HeightStore, client GRPC, cinn chan hBTx) {
	defer wg.Done()
	for in := range cinn {
		b, txs, err := blockAndTx(ctx, logger, client, in.Height)
		if err != nil {
			in.Ch <- cStructs.OutResp{
				ID:    b.ID,
				Error: err,
				Type:  "Error",
			}
			return
		}
		if err := hstore.StoreBlocks(ctx, []structs.BlockWithMeta{b}); err != nil {
			in.Ch <- cStructs.OutResp{
				ID:    b.ID,
				Error: err,
				Type:  "Error",
			}
			return
		}

		if err := hstore.StoreTransactions(ctx, txs); err != nil {
			in.Ch <- cStructs.OutResp{
				ID:    b.ID,
				Error: err,
				Type:  "Error",
			}
			return
		}

		if err := hstore.ConfirmHeights(ctx, []uint64{in.Height}); err != nil {
		}

	}
}

type hBTx struct {
	Height uint64
	Last   bool
	Ch     chan cStructs.OutResp
}

// getRange gets given range of blocks and transactions
func getRange(ctx context.Context, logger *zap.Logger, client GRPC, hr structs.HeightRange) (h structs.Heights, err error) {
	defer logger.Sync()

	errored := make(chan bool, 7)
	defer close(errored)

	wg := &sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go asyncBlockAndTx(ctx, logger, wg, client)
	}
	go populateRange(chIn, chOut, hr, errored)

	return err
}

/*
// GetLatest gets latest transactions and blocks.
// It gets latest transaction, then diff it with
func (ic *IndexerClient) GetLatest(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client GRPC) {
	timer := metrics.NewTimer(getLatestDuration)
	defer timer.ObserveDuration()

	ldr := &structs.LatestDataRequest{}
	err := json.Unmarshal(tr.Payload, ldr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{Id: tr.Id, Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"}, Final: true})
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// (lukanus): Get latest block (height = 0)
	block, err := client.GetBlock(sCtx, structs.HeightHash{})
	if err != nil {
		stream.Send(cStructs.TaskResponse{Id: tr.Id, Error: cStructs.TaskError{Msg: "Error getting block data " + err.Error()}, Final: true})
		return
	}

	hr := getLastHeightRange(ldr.LastHeight, ic.maximumHeightsToGet, block.Height)

	out := make(chan cStructs.OutResp, page*2+1)
	fin := make(chan bool, 2)

	// (lukanus): in separate goroutine take transaction format wrap it in transport message and send
	go sendResp(sCtx, tr.Id, out, ic.logger, stream, fin)

	ic.logger.Debug("[COSMOS-CLIENT] Getting Range", zap.Stringer("taskID", tr.Id), zap.Uint64("start", hr.StartHeight), zap.Uint64("end", hr.EndHeight))
	if err := getRange(sCtx, ic.logger, ic.grpc, hr, out); err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: err.Error()},
			Final: true,
		})
		ic.logger.Error("[COSMOS-CLIENT] Error getting range (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
		close(out)
		return
	}
	close(out)

	for {
		select {
		case <-sCtx.Done():
			return
		case <-fin:
			ic.logger.Debug("[COSMOS-CLIENT] Finished sending all", zap.Stringer("taskID", tr.Id), zap.Any("heights", hr))
			return
		}
	}
}

// getLastHeightRange - based current state
func getLastHeightRange(lastKnownHeight, maximumHeightsToGet, lastBlockFromNetwork uint64) structs.HeightRange {
	// (lukanus): When nothing is scraped we want to get only X number of last requests
	if lastKnownHeight == 0 {
		lastX := lastBlockFromNetwork - maximumHeightsToGet
		if lastX > 0 {
			return structs.HeightRange{
				StartHeight: lastX,
				EndHeight:   lastBlockFromNetwork,
			}
		}
	}

	if maximumHeightsToGet < lastBlockFromNetwork-lastKnownHeight {
		return structs.HeightRange{
			StartHeight: lastBlockFromNetwork - maximumHeightsToGet,
			EndHeight:   lastBlockFromNetwork,
		}
	}

	return structs.HeightRange{
		StartHeight: lastKnownHeight,
		EndHeight:   lastBlockFromNetwork,
	}
}


*/
