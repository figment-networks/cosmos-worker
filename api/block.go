package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"

	"github.com/figment-networks/indexer-manager/structs"
)

// BlocksMap map of blocks to control block map
// with extra summary of number of transactions
type BlocksMap struct {
	sync.Mutex
	Blocks map[uint64]structs.Block
	NumTxs uint64
}

// BlockErrorPair to wrap error response
type BlockErrorPair struct {
	Height uint64
	Block  structs.Block
	Err    error
}

// GetBlock fetches most recent block from chain
func (c Client) GetBlock(ctx context.Context, params structs.HeightHash) (block structs.Block, er error) {
	var ok bool
	if params.Height != 0 {
		block, ok = c.Sbc.Get(params.Height)
		if ok {
			blockCacheEfficiencyHit.Inc()
			return block, nil
		}
		blockCacheEfficiencyMissed.Inc()
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return block, err
	}

	n := time.Now()
	if params.Height > 0 {
		bbh, err := c.tmServiceClient.GetBlockByHeight(ctx, &tmservice.GetBlockByHeightRequest{Height: int64(params.Height)})
		if err != nil {
			rawRequestDuration.WithLabels("GetBlockByHeight", "error").Observe(time.Since(n).Seconds())
			return block, err
		}

		rawRequestDuration.WithLabels("GetBlockByHeight", "ok").Observe(time.Since(n).Seconds())
		block = structs.Block{
			Hash:                 bbh.BlockId.String(),
			Height:               uint64(bbh.Block.Header.Height),
			Time:                 bbh.Block.Header.Time,
			ChainID:              bbh.Block.Header.ChainID,
			NumberOfTransactions: uint64(len(bbh.Block.Data.Txs)),
		}
	} else {
		lb, err := c.tmServiceClient.GetLatestBlock(ctx, &tmservice.GetLatestBlockRequest{})
		if err != nil {
			rawRequestDuration.WithLabels("GetBlockByHeight", "error").Observe(time.Since(n).Seconds())
			return block, err
		}
		rawRequestDuration.WithLabels("GetBlockByHeight", "ok").Observe(time.Since(n).Seconds())

		block = structs.Block{
			Hash:                 string(lb.BlockId.Hash),
			Height:               uint64(lb.Block.Header.Height),
			Time:                 lb.Block.Header.Time,
			ChainID:              lb.Block.Header.ChainID,
			NumberOfTransactions: uint64(len(lb.Block.Data.Txs)),
		}
	}

	c.Sbc.Add(block)

	return block, nil
}

func (c Client) GetBlockAsync(ctx context.Context, in chan uint64, out chan<- BlockErrorPair) {
	for height := range in {
		b, err := c.GetBlock(ctx, structs.HeightHash{Height: height})

		out <- BlockErrorPair{
			Height: height,
			Block:  b,
			Err:    err,
		}
	}

}

func (c Client) GetBlocksMeta(ctx context.Context, params structs.HeightRange, blocks *BlocksMap, end chan<- error) {

	total := params.EndHeight - params.StartHeight
	if total == 0 {
		total = 1
	}

	for i := uint64(0); i < total; i++ {
		block, err := c.GetBlock(ctx, structs.HeightHash{Height: uint64(params.StartHeight) + i - 1})
		if err != nil {
			end <- fmt.Errorf("error fetching block: %d %w ", uint64(params.StartHeight)+i-1, err)
			return
		}
		blocks.Lock()
		blocks.Blocks[block.Height] = block
		blocks.Unlock()
	}

	end <- nil
}

/*
// GetBlocksMeta fetches block metadata from given range of blocks
func (c Client) GetBlocksMeta(ctx context.Context, params structs.HeightRange, blocks *BlocksMap, end chan<- error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/blockchain", nil)
	if err != nil {
		end <- err
		return
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	if params.StartHeight > 0 {
		q.Add("minHeight", strconv.FormatUint(params.StartHeight, 10))
	}

	if params.EndHeight > 0 {
		q.Add("maxHeight", strconv.FormatUint(params.EndHeight, 10))
	}
	req.URL.RawQuery = q.Encode()

	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		end <- err
		return
	}

	n := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		end <- err
		return
	}
	rawRequestDuration.WithLabels("/blockchain", resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockchainResponse
	if err = decoder.Decode(&result); err != nil {
		end <- err
		return
	}

	if result.Error.Message != "" {
		end <- fmt.Errorf("error fetching block: %s ", result.Error.Message)
		return
	}

	blocks.Lock()
	for _, meta := range result.Result.BlockMetas {

		bTime, _ := time.Parse(time.RFC3339Nano, meta.Header.Time)
		uHeight, _ := strconv.ParseUint(meta.Header.Height, 10, 64)
		numTxs, _ := strconv.ParseUint(meta.Header.NumTxs, 10, 64)

		block := structs.Block{
			Hash:                 meta.BlockID.Hash,
			Height:               uHeight,
			ChainID:              meta.Header.ChainID,
			Time:                 bTime,
			NumberOfTransactions: numTxs,
		}
		blocks.NumTxs += numTxs
		blocks.Blocks[block.Height] = block
	}
	blocks.Unlock()

	end <- nil
	return
}
*/
