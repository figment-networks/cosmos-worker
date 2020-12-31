package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/figment-networks/indexer-manager/structs"
)

var queryRewardsEndpoint = fmt.Sprintf("custom/distribution/%s", types.QueryDelegatorTotalRewards)

// GetReward fetches total rewards for delegator account
func (c *Client) GetReward(ctx context.Context, req structs.HeightAccount) (rewards []structs.TransactionAmount, err error) {
	c.cliCtx = c.cliCtx.WithHeight(int64(req.Height))

	addr, err := getAddr(req.Account)
	if err != nil {
		return
	}

	params := types.NewQueryDelegatorParams(addr)
	bz, err := c.cliCtx.Codec.MarshalJSON(params)
	if err != nil {
		return
	}
	fmt.Println("[GetReward] req height=", req.Height)

	res, _, err := c.cliCtx.QueryWithData(queryRewardsEndpoint, bz)
	if err != nil {
		return
	}

	var data types.QueryDelegatorTotalRewardsResponse
	err = json.Unmarshal(res, &data)
	if err != nil {
		return
	}

	for _, coin := range data.Total {
		rewards = append(rewards, structs.TransactionAmount{
			Text:     coin.Amount.String(),
			Numeric:  coin.Amount.BigInt(),
			Currency: coin.Denom,
			Exp:      sdk.Precision,
		})
	}
	return
}

func getAddr(addrString string) (addr []byte, err error) {
	// try hex, then bech32
	var err1 error
	addr, err1 = hex.DecodeString(addrString)
	if err1 != nil {
		var err2 error
		addr, err2 = sdk.AccAddressFromBech32(addrString)
		if err2 != nil {
			var err3 error
			addr, err3 = sdk.ValAddressFromBech32(addrString)
			if err3 != nil {
				err = fmt.Errorf("expected hex or bech32. Got errors: hex: %v, bech32 acc: %v, bech32 val: %v", err1, err2, err3)
				return
			}
		}
	}
	return
}
