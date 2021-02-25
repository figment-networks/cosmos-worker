package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/figment-networks/cosmos-worker/api"
	"github.com/figment-networks/cosmos-worker/client"
	"github.com/figment-networks/indexer-manager/structs"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
)

func TestGetBlock(t *testing.T) {
	type args struct {
		address string
		from    big.Int
		to      big.Int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				address: "localhost:9090",
				from:    *big.NewInt(10880000),
				to:      *big.NewInt(10890000),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			zl := zaptest.NewLogger(t)

			api.InitMetrics()
			conn, err := grpc.Dial(tt.args.address, grpc.WithInsecure())
			require.NoError(t, err)
			cli := api.NewClient(zl, conn, 10, "", "")
			end := make(chan error, 10)
			defer close(end)

			bm := &api.BlocksMap{
				Blocks: make(map[uint64]structs.Block),
			}

			err = cli.GetBlocksMeta(ctx, structs.HeightRange{StartHeight: 300, EndHeight: 450}, bm)
			require.NoError(t, err)
			for _, b := range bm.Blocks {
				if b.NumberOfTransactions > 0 {
					txs, err := cli.SearchTx(ctx, structs.HeightHash{Height: b.Height}, b, 1)
					require.NoError(t, err)
					t.Logf("txs %+v", txs)
				}
			}
		})
	}
}

func TestGetResponseConsistency(t *testing.T) {
	type args struct {
		address string
		hRange  structs.HeightRange
		reqsec  int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "test1",
		args: args{
			address: "localhost:9090",
			hRange:  structs.HeightRange{StartHeight: 1, EndHeight: 1},
			reqsec:  300,
		},
	}, {
		name: "test2",
		args: args{
			address: "localhost:9090",
			hRange:  structs.HeightRange{StartHeight: 1, EndHeight: 1000},
			reqsec:  300,
		},
	}, {
		name: "test3",
		args: args{
			address: "localhost:9090",
			hRange:  structs.HeightRange{StartHeight: 1930, EndHeight: 3900},
			reqsec:  300,
		},
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zl := zaptest.NewLogger(t)

			ctx := context.Background()
			api.InitMetrics()
			conn, err := grpc.Dial(tt.args.address, grpc.WithInsecure())
			require.NoError(t, err)
			apiClient := api.NewClient(zl, conn, tt.args.reqsec, "", "")
			workerClient := client.NewIndexerClient(ctx, zl, apiClient, uint64(1000))

			sr := newSendRegistry()
			trp, _ := json.Marshal(tt.args.hRange)
			workerClient.GetTransactions(ctx, cStructs.TaskRequest{Id: uuid.New(), Payload: trp}, sr, apiClient)
			require.NoError(t, err)
			cbr := sr.CheckForBlockRange(tt.args.hRange.StartHeight, tt.args.hRange.EndHeight)
			t.Log(sr.Summary())
			t.Logf("missing records %v", cbr)
			require.Empty(t, cbr)

			conn.Close()

		})
	}
}

type sendRegistry struct {
	blocks       map[uint64]cStructs.TaskResponse
	transactions map[uint64][]cStructs.TaskResponse
	ends         []cStructs.TaskResponse
	errors       []cStructs.TaskResponse
	other        []cStructs.TaskResponse
}

func newSendRegistry() *sendRegistry {
	return &sendRegistry{
		blocks:       make(map[uint64]cStructs.TaskResponse),
		transactions: make(map[uint64][]cStructs.TaskResponse),
	}
}
func (sR *sendRegistry) Send(tr cStructs.TaskResponse) error {

	if tr.Error.Msg != "" {
		sR.errors = append(sR.errors, tr)
	}

	switch tr.Type {
	case "Block":
		var b *structs.Block
		err := json.Unmarshal(tr.Payload, &b)
		if err != nil {
			return err
		}
		sR.blocks[b.Height] = tr
	case "Transaction":
		var t *structs.Transaction
		err := json.Unmarshal(tr.Payload, &t)
		if err != nil {
			return err
		}
		txs, ok := sR.transactions[t.Height]
		if !ok {
			txs = []cStructs.TaskResponse{}
		}
		txs = append(txs, tr)
		sR.transactions[t.Height] = txs
	case "END":
		sR.ends = append(sR.ends, tr)
	default:
		sR.other = append(sR.other, tr)
	}
	return nil
}

func (sR *sendRegistry) CheckForBlockRange(start, end uint64) (missing []uint64) {
	for i := start; i < end+1; i++ {
		if _, ok := sR.blocks[i]; !ok {
			missing = append(missing, i)
		}
	}
	return missing
}

func (sR *sendRegistry) Summary() string {
	return fmt.Sprintf("Finished with blocks: %d transactions: %d errors: %d ends: %d other: %d", len(sR.blocks), len(sR.transactions), len(sR.errors), len(sR.ends), len(sR.other))
}

func TestGetDelegatorReward(t *testing.T) {
	lcdAddr := "https://cosmoshub-4--lcd--archive.datahub.figment.io"
	dataHubKey := "" // set your api key before testing
	tests := []struct {
		name       string
		lcdAddr    string
		dataHubKey string
		args       structs.HeightAccount
		resultText string
		wantErr    bool
	}{
		{
			name:       "wrong delegator address syntax",
			lcdAddr:    lcdAddr,
			dataHubKey: dataHubKey,
			args: structs.HeightAccount{
				Account: "wrong delegator address",
			},
			wantErr: true,
		},
		{
			name:       "present delegator first reward at height",
			lcdAddr:    lcdAddr,
			dataHubKey: dataHubKey,
			// see: https://www.mintscan.io/cosmos/account/cosmos1nlx3qm563gcr0xnzdtynj00japy7w04pmmljt0
			args: structs.HeightAccount{
				Account: "cosmos1nlx3qm563gcr0xnzdtynj00japy7w04pmmljt0",
				Height:  5217493,
			},
			resultText: "0.080405949465470000",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			zl := zaptest.NewLogger(t)
			capi := api.NewClient(zl, nil, 10, tt.lcdAddr, tt.dataHubKey)
			resp, err := capi.GetReward(ctx, tt.args)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				fmt.Println(resp.Height)
				require.Equal(t, resp.Rewards[0].Text, tt.resultText)
			}
		})
	}
}
