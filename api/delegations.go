package api

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"google.golang.org/grpc/metadata"

	"github.com/figment-networks/indexer-manager/structs"
)

var unbondingDenom = "uatom"

// GetAccountDelegations fetches account delegations
func (c *Client) GetAccountDelegations(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountDelegationsResponse, err error) {
	resp.Height = params.Height

	delResp, err := c.stakingClient.DelegatorDelegations(metadata.AppendToOutgoingContext(ctx, grpctypes.GRPCBlockHeightHeader, strconv.FormatUint(params.Height, 10)),
		&types.QueryDelegatorDelegationsRequest{DelegatorAddr: params.Account})
	if err != nil {
		return resp, fmt.Errorf("[COSMOS-API] Error fetching delegations: %w", err)
	}

	for _, dr := range delResp.DelegationResponses {
		resp.Delegations = append(resp.Delegations,
			structs.Delegation{
				Delegator: dr.Delegation.DelegatorAddress,
				Validator: structs.Validator(dr.Delegation.ValidatorAddress),
				Shares: structs.TransactionAmount{
					Numeric: dr.Delegation.Shares.BigInt(),
					Exp:     sdk.Precision,
				},
				Balance: structs.TransactionAmount{
					Text:     dr.Balance.Amount.String(),
					Numeric:  dr.Balance.Amount.BigInt(),
					Currency: dr.Balance.Denom,
				},
			},
		)
	}

	return resp, err
}

// GetAccountUnbondingDelegations fetches account delegations
func (c *Client) GetAccountUnbondingDelegations(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountUnbondingResponse, err error) {
	resp.Height = params.Height

	delResp, err := c.stakingClient.DelegatorUnbondingDelegations(metadata.AppendToOutgoingContext(ctx, grpctypes.GRPCBlockHeightHeader, strconv.FormatUint(params.Height, 10)),
		&types.QueryDelegatorUnbondingDelegationsRequest{DelegatorAddr: params.Account})
	if err != nil {
		return resp, fmt.Errorf("[COSMOS-API] Error fetching unbonding delegations: %w", err)
	}

	for _, dr := range delResp.UnbondingResponses {
		for _, entry := range dr.Entries {
			resp.UnbondingDelegations = append(resp.UnbondingDelegations, structs.UnbondingDelegation{
				Delegator:      dr.DelegatorAddress,
				Validator:      structs.Validator(dr.ValidatorAddress),
				CreationHeight: entry.CreationHeight,
				CompletionTime: entry.CompletionTime,
				InitialBalance: structs.TransactionAmount{
					Currency: unbondingDenom,
					Numeric:  entry.InitialBalance.BigInt(),
				},
				Balance: structs.TransactionAmount{
					Currency: unbondingDenom,
					Numeric:  entry.Balance.BigInt(),
				},
			})
		}
	}

	return resp, err
}
