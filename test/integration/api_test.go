package integration

import (
	"context"
	"math/big"
	"testing"

	"github.com/figment-networks/cosmos-worker/api"
	"github.com/figment-networks/indexer-manager/structs"
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
			cli := api.NewClient(zl, conn, 10)
			end := make(chan error, 10)
			defer close(end)

			bm := &api.BlocksMap{
				Blocks: make(map[uint64]structs.Block),
			}
			cli.GetBlocksMeta(ctx, structs.HeightRange{StartHeight: 200, EndHeight: 420}, bm, end)
			for _, b := range bm.Blocks {
				if b.NumberOfTransactions > 0 {
					txs, err := cli.SearchTx(ctx, structs.HeightHash{Height: b.Height}, b)
					require.NoError(t, err)
					t.Logf("txs %+v", txs)
				}
			}

			//log.Println("a,er", a, er)
			//
			/*
				if err := eAPI.ParseLogs(ctx, ccs, tt.args.from, tt.args.to); err != nil {
					t.Error(err)
					return
				}*/
		})
	}
}
