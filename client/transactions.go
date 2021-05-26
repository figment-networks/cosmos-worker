package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/figment-networks/indexer-manager/structs"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
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

	ic.logger.Debug("[COSMOS-CLIENT] Getting Range", zap.Stringer("taskID", tr.Id), zap.Uint64("start", hr.StartHeight), zap.Uint64("end", hr.EndHeight))

	heights, err := ic.Reqester.GetRange(ctx, *hr)
	resp := &cStructs.TaskResponse{
		Id:    tr.Id,
		Type:  "Heights",
		Final: true,
	}
	if heights.NumberOfHeights > 0 {
		resp.Payload, _ = json.Marshal(heights)
	}
	if err != nil {
		resp.Error = cStructs.TaskError{Msg: err.Error()}
		ic.logger.Error("[COSMOS-CLIENT] Error getting range (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
		if err := stream.Send(*resp); err != nil {
			ic.logger.Error("[COSMOS-CLIENT] Error sending message (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
		}
		return
	}
	if err := stream.Send(*resp); err != nil {
		ic.logger.Error("[COSMOS-CLIENT] Error sending message (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
	}
	ic.logger.Debug("[COSMOS-CLIENT] Finished sending all", zap.Stringer("taskID", tr.Id), zap.Any("heights", hr))
}

func (ic *IndexerClient) BlockAndTx(ctx context.Context, height uint64) (blockWM structs.BlockWithMeta, txsWM []structs.TransactionWithMeta, err error) {
	defer ic.logger.Sync()
	ic.logger.Debug("[COSMOS-CLIENT] Getting height", zap.Uint64("block", height))

	hSess, err := ic.storeClient.GetSession(ctx)
	if err != nil {
		return blockWM, nil, err
	}

	blockWM = structs.BlockWithMeta{Network: "cosmos", Version: "0.0.1"}
	blockWM.Block, err = ic.grpc.GetBlock(ctx, structs.HeightHash{Height: uint64(height)})
	blockWM.ChainID = blockWM.Block.ChainID
	if err != nil {
		ic.logger.Error("[COSMOS-CLIENT] Err Getting block", zap.Uint64("block", height), zap.Error(err), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
		return blockWM, nil, fmt.Errorf("error fetching block: %d %w ", uint64(height), err)
	}
	if err := hSess.StoreBlocks(ctx, []structs.BlockWithMeta{blockWM}); err != nil {
		return blockWM, nil, err
	}

	if blockWM.Block.NumberOfTransactions > 0 {
		ic.logger.Debug("[COSMOS-CLIENT] Getting txs", zap.Uint64("block", height), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
		var txs []structs.Transaction
		txs, err = ic.grpc.SearchTx(ctx, structs.HeightHash{Height: height}, blockWM.Block, page)
		for _, t := range txs {
			txsWM = append(txsWM, structs.TransactionWithMeta{Network: "cosmos", ChainID: t.ChainID, Version: "0.0.1", Transaction: t})
		}
		if len(txsWM) > 0 {
			if err := hSess.StoreTransactions(ctx, txsWM); err != nil {
				return blockWM, txsWM, err
			}
		}
		ic.logger.Debug("[COSMOS-CLIENT] txErr Getting txs", zap.Uint64("block", height), zap.Error(err), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
	}

	if err := hSess.ConfirmHeights(ctx, []structs.BlockWithMeta{blockWM}); err != nil {
		return blockWM, txsWM, err
	}
	ic.logger.Debug("[COSMOS-CLIENT] Got block", zap.Uint64("block", height), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
	return blockWM, txsWM, err
}
