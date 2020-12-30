package integration

import (
	"context"
	"log"
	"testing"

	"github.com/figment-networks/cosmos-worker/api"
	"github.com/figment-networks/indexer-manager/structs"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	crpc "github.com/figment-networks/cosmos-worker/rpc"
)

func TestGetReward(t *testing.T) {
	type args struct {
		address string
		hs      structs.HeightAccount
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				address: "http://cosmoshub-3--rpc--archive.datahub.figment.io:80",
				hs:      structs.HeightAccount{Height: 4583857, Account: "cosmos1tflk30mq5vgqjdly92kkhhq3raev2hnzldd74z"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			zl := zaptest.NewLogger(t)

			api.InitMetrics()
			cli, err := crpc.NewClient(tt.args.address, "", "cosmoshub-3", zl, 10)
			require.NoError(t, err)

			r, err := cli.GetReward(ctx, structs.HeightAccount{})
			require.NoError(t, err)
			log.Println("r", r)
		})
	}
}
