package api

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func mapDistributionWithdrawValidatorCommissionToSub(msg []byte) (se shared.SubsetEvent, er error) {
	wvc := &distribution.MsgWithdrawValidatorCommission{}
	if err := proto.Unmarshal(msg, wvc); err != nil {
		return se, errors.New("Not a distribution type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"withdraw_validator_commission"},
		Module: "distribution",
		Node:   map[string][]shared.Account{"validator": {{ID: wvc.ValidatorAddress}}},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wvc.ValidatorAddress},
		}},
	}, nil
}

func mapDistributionSetWithdrawAddressToSub(msg []byte) (se shared.SubsetEvent, er error) {
	swa := &distribution.MsgSetWithdrawAddress{}
	if err := proto.Unmarshal(msg, swa); err != nil {
		return se, errors.New("Not a set_withdraw_address type" + err.Error())
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

func mapDistributionWithdrawDelegatorRewardToSub(msg []byte) (se shared.SubsetEvent, er error) {
	wdr := &distribution.MsgWithdrawDelegatorReward{}
	if err := proto.Unmarshal(msg, wdr); err != nil {
		return se, errors.New("Not a withdraw_validator_commission type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"withdraw_delegator_reward"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: wdr.DelegatorAddress}},
			"validator": {{ID: wdr.ValidatorAddress}},
		},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wdr.ValidatorAddress},
		}},
	}, nil
}

func mapDistributionFundCommunityPoolToSub(msg []byte) (se shared.SubsetEvent, er error) {
	fcp := &distribution.MsgFundCommunityPool{}
	if err := proto.Unmarshal(msg, fcp); err != nil {
		return se, errors.New("Not a fund_community_pool type" + err.Error())
	}

	evt, err := distributionProduceEvTx(fcp.Depositor, fcp.Amount)
	return shared.SubsetEvent{
		Type:   []string{"fund_community_pool"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"depositor": {{ID: fcp.Depositor}},
		},
		Sender: []shared.EventTransfer{evt},
	}, err

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
