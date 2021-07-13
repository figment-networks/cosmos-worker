package mapper

import (
	"fmt"
	"strconv"

	shared "github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	client "github.com/gravity-devs/liquidity/x/liquidity/types"
)

// see types https://github.com/Gravity-Devs/liquidity/blob/44220af8ebd5b664768b4098a2159b75ca02df8a/x/liquidity/spec/04_messages.md

// TendermintCreatePool transforms liquidity.MsgCreatePool sdk messages to SubsetEvent
func TendermintCreatePool(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgCreatePool{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a create_pool type: %w", err)
	}

	se = shared.SubsetEvent{
		Type:   []string{"create_pool"},
		Module: "tendermint_liquidity",
		Node: map[string][]shared.Account{
			"pool_creator_address": {{ID: m.PoolCreatorAddress}},
		},
		Additional: map[string][]string{
			"pool_type_id": {strconv.FormatUint(uint64(m.PoolTypeId), 10)},
		},
	}

	se.Amount = map[string]shared.TransactionAmount{}
	for i, coin := range m.DepositCoins {
		se.Amount[fmt.Sprintf("deposit_coin_%d", i)] = shared.TransactionAmount{
			Currency: coin.Denom,
			Numeric:  coin.Amount.BigInt(),
			Text:     coin.String(),
		}
	}

	return se, nil
}

// TendermintDepositWithinBatch transforms liquidity.MsgDepositWithinBatch sdk messages to SubsetEvent
func TendermintDepositWithinBatch(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgDepositWithinBatch{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a deposit_within_batch type: %w", err)
	}

	se = shared.SubsetEvent{
		Type:   []string{"deposit_within_batch"},
		Module: "tendermint_liquidity",
		Node: map[string][]shared.Account{
			"depositor_address": {{ID: m.DepositorAddress}},
		},
		Additional: map[string][]string{
			"pool_id": {strconv.FormatUint(m.PoolId, 10)},
		},
	}

	se.Amount = map[string]shared.TransactionAmount{}
	for i, coin := range m.DepositCoins {
		se.Amount[fmt.Sprintf("deposit_coin_%d", i)] = shared.TransactionAmount{
			Currency: coin.Denom,
			Numeric:  coin.Amount.BigInt(),
			Text:     coin.String(),
		}
	}
	return se, nil
}

// TendermintWithdrawWithinBatch transforms liquidity.MsgWithdrawWithinBatch sdk messages to SubsetEvent
func TendermintWithdrawWithinBatch(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgWithdrawWithinBatch{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a withdraw_within_batch type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"withdraw_within_batch"},
		Module: "tendermint_liquidity",
		Node: map[string][]shared.Account{
			"withdrawer_address": {{ID: m.WithdrawerAddress}},
		},
		Amount: map[string]shared.TransactionAmount{
			"pool_coin": {
				Currency: m.PoolCoin.Denom,
				Numeric:  m.PoolCoin.Amount.BigInt(),
				Text:     m.PoolCoin.String(),
			},
		},
		Additional: map[string][]string{
			"pool_id": {strconv.FormatUint(m.PoolId, 10)},
		},
	}, nil
}

// TendermintSwapWithinBatch transforms liquidity.MsgSwapWithinBatch sdk messages to SubsetEvent
func TendermintSwapWithinBatch(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgSwapWithinBatch{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a swap_within_batch type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"swap_within_batch"},
		Module: "tendermint_liquidity",
		Node: map[string][]shared.Account{
			"swap_requester": {{ID: m.SwapRequesterAddress}},
		},
		Amount: map[string]shared.TransactionAmount{
			"demand_coin_denom": {
				Currency: m.DemandCoinDenom,
			},
			"order_price": {
				Numeric: m.OrderPrice.BigInt(),
				Text:    m.OrderPrice.String(),
				Exp:     sdk.Precision,
			},
			"offer_coin": {
				Currency: m.OfferCoin.Denom,
				Numeric:  m.OfferCoin.Amount.BigInt(),
				Text:     m.OfferCoin.String(),
			},
			"offer_coin_fee": {
				Currency: m.OfferCoinFee.Denom,
				Numeric:  m.OfferCoinFee.Amount.BigInt(),
				Text:     m.OfferCoinFee.String(),
			},
		},
		Additional: map[string][]string{
			"pool_id":      {strconv.FormatUint(m.PoolId, 10)},
			"swap_type_id": {strconv.FormatUint(uint64(m.SwapTypeId), 10)},
		},
	}, nil
}
