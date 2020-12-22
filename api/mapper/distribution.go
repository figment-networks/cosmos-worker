package mapper

import (
	"errors"
	"math/big"

	"github.com/figment-networks/cosmos-worker/api/types"
	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

var zero big.Int

// DistributionWithdrawValidatorCommissionToSub transforms distribution.MsgWithdrawValidatorCommission sdk messages to SubsetEvent
func DistributionWithdrawValidatorCommissionToSub(msg sdk.Msg, logf types.LogFormat) (se shared.SubsetEvent, err error) {
	wvc, ok := msg.(distribution.MsgWithdrawValidatorCommission)
	if !ok {
		return se, errors.New("Not a withdraw_validator_commission type")
	}

	se = shared.SubsetEvent{
		Type:   []string{"withdraw_validator_commission"},
		Module: "distribution",
		Node:   map[string][]shared.Account{"validator": {{ID: wvc.ValidatorAddress}}},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wvc.ValidatorAddress},
		}},
	}

	err = produceTransfers(&se, TransferTypeSend, logf)
	return se, err
}

// DistributionSetWithdrawAddressToSub transforms distribution.MsgSetWithdrawAddress sdk messages to SubsetEvent
func DistributionSetWithdrawAddressToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	swa, ok := msg.(distribution.MsgSetWithdrawAddress)
	if !ok {
		return se, errors.New("Not a set_withdraw_address type")
	}

	return shared.SubsetEvent{
		Type:   []string{"set_withdraw_address"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: swa.DelegatorAddress}},
			"withdraw":  {{ID: swa.WithdrawAddress}},
		},
	}, nil
}

// DistributionWithdrawDelegatorRewardToSub transforms distribution.MsgWithdrawDelegatorReward sdk messages to SubsetEvent
func DistributionWithdrawDelegatorRewardToSub(msg sdk.Msg, logf types.LogFormat) (se shared.SubsetEvent, err error) {
	wdr, ok := msg.(distribution.MsgWithdrawDelegatorReward)
	if !ok {
		return se, errors.New("Not a withdraw_validator_commission type")
	}
	se = shared.SubsetEvent{
		Type:   []string{"withdraw_delegator_reward"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: wdr.DelegatorAddress}},
			"validator": {{ID: wdr.ValidatorAddress}},
		},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wdr.DelegatorAddress.String()},
		}},
	}

	err = produceTransfers(&se, TransferTypeReward, logf)
	return se, err
}

// DistributionFundCommunityPoolToSub transforms distributiontypes.MsgFundCommunityPool sdk messages to SubsetEvent
func DistributionFundCommunityPoolToSub(msg sdk.Msg, logf types.LogFormat) (se shared.SubsetEvent, er error) {
	fcp, ok := msg.(distributiontypes.MsgFundCommunityPool)
	if !ok {
		return se, errors.New("Not a withdraw_validator_commission type")
	}

	evt, err := distributionProduceEvTx(fcp.Depositor, fcp.Amount)
	se = shared.SubsetEvent{
		Type:   []string{"fund_community_pool"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"depositor": {{ID: fcp.Depositor}},
		},
		Sender: []shared.EventTransfer{evt},
	}
	err = produceTransfers(&se, TransferTypeReward, logf)
	return se, err
}

func distributionProduceEvTx(account string, coins types.Coins) (evt shared.EventTransfer, err error) {

	evt = shared.EventTransfer{
		Account: shared.Account{ID: account},
	}
	if len(coins) > 0 {
		evt.Amounts = []shared.TransactionAmount{}
		for _, coin := range coins {
			txa := shared.TransactionAmount{
				Currency: coin.Denom,
				Text:     coin.Amount.String(),
			}

			txa.Numeric.Set(coin.Amount.BigInt())
			evt.Amounts = append(evt.Amounts, txa)
		}
	}

	return evt, nil
}
