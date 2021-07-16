package api

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/figment-networks/indexing-engine/structs"
	"google.golang.org/grpc/metadata"
)

type responseWithHeight struct {
	Height string                                   `json:"height"`
	Result types.QueryDelegatorTotalRewardsResponse `json:"result"`
}

const maxRetries = 3

// GetReward fetches total rewards for delegator account
func (c *Client) GetReward(
	ctx context.Context,
	params structs.HeightAccount,
	chainID string,
) (resp structs.GetUnclaimedRewardResponse, err error) {
	resp.Height = params.Height

	valResp, err := c.distributionClient.DelegatorValidators(metadata.AppendToOutgoingContext(ctx, grpctypes.GRPCBlockHeightHeader, strconv.FormatUint(params.Height, 10)),
		&types.QueryDelegatorValidatorsRequest{DelegatorAddress: params.Account})
	if err != nil {
		return resp, fmt.Errorf("[COSMOS-API] Error fetching validators: %w", err)
	}

	resp.UnclaimedRewards = make([]structs.UnclaimedReward, 0, len(valResp.Validators))

	// get rewards delegator earned from each of its validators
	for _, val := range valResp.Validators {
		delResp, err := c.distributionClient.DelegationRewards(metadata.AppendToOutgoingContext(ctx, grpctypes.GRPCBlockHeightHeader, strconv.FormatUint(params.Height, 10)),
			&types.QueryDelegationRewardsRequest{DelegatorAddress: params.Account, ValidatorAddress: val})
		if err != nil {
			return resp, fmt.Errorf("[COSMOS-API] Error fetching delegation rewards: %w", err)
		}

		delRewards := delResp.GetRewards()

		// translate each amount to RewardAmount
		valRewards := make([]structs.RewardAmount, 0, len(delRewards))
		for _, reward := range delRewards {
			valRewards = append(valRewards,
				structs.RewardAmount{
					Text:     reward.Amount.String(),
					Numeric:  reward.Amount.BigInt(),
					Currency: reward.Denom,
					Exp:      sdk.Precision,
				},
			)
		}

		reward := structs.UnclaimedReward{
			Account:         params.Account,
			ChainID:         chainID,
			Height:          params.Height,
			Network:         "cosmos",
			UnclaimedReward: valRewards,
			Validator:       val,
		}

		resp.UnclaimedRewards = append(resp.UnclaimedRewards, reward)
	}

	return resp, err
}
