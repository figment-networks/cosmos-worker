package api

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/figment-networks/indexer-manager/structs"
	"google.golang.org/grpc/metadata"
)

type responseWithHeight struct {
	Height string                                   `json:"height"`
	Result types.QueryDelegatorTotalRewardsResponse `json:"result"`
}

const maxRetries = 3

// GetReward fetches total rewards for delegator account
func (c *Client) GetReward(ctx context.Context, params structs.HeightAccount) (resp structs.GetRewardResponse, err error) {
	resp.Height = params.Height

	delResp, err := c.distributionClient.DelegationTotalRewards(metadata.AppendToOutgoingContext(ctx, grpctypes.GRPCBlockHeightHeader, strconv.FormatUint(params.Height, 10)),
		&types.QueryDelegationTotalRewardsRequest{DelegatorAddress: params.Account})
	if err != nil {
		return resp, fmt.Errorf("[COSMOS-API] Error fetching rewards: %w", err)
	}

	for _, reward := range delResp.Total {
		resp.Rewards = append(resp.Rewards,
			structs.TransactionAmount{
				Text:     reward.Amount.String(),
				Numeric:  reward.Amount.BigInt(),
				Currency: reward.Denom,
				Exp:      sdk.Precision,
			},
		)
	}

	return resp, err
}
