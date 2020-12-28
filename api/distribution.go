package api

import (
	"errors"
	"fmt"
	"math/big"

	shared "github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

var zero big.Int

func mapDistributionWithdrawValidatorCommissionToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	wvc, ok := msg.(distribution.MsgWithdrawValidatorCommission)
	if !ok {
		return se, errors.New("Not a withdraw_validator_commission type")
	}

	return shared.SubsetEvent{
		Type:   []string{"withdraw_validator_commission"},
		Module: "distribution",
		Node:   map[string][]shared.Account{"validator": {{ID: wvc.ValidatorAddress.String()}}},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wvc.ValidatorAddress.String()},
		}},
	}, nil
}

func mapDistributionSetWithdrawAddressToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	swa, ok := msg.(distribution.MsgSetWithdrawAddress)
	if !ok {
		return se, errors.New("Not a set_withdraw_address type")
	}

	return shared.SubsetEvent{
		Type:   []string{"set_withdraw_address"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: swa.DelegatorAddress.String()}},
			"withdraw":  {{ID: swa.WithdrawAddress.String()}},
		},
	}, nil
}

func mapDistributionWithdrawDelegatorRewardToSub(msg sdk.Msg, logf LogFormat) (se shared.SubsetEvent, er error) {
	wdr, ok := msg.(distribution.MsgWithdrawDelegatorReward)
	if !ok {
		return se, errors.New("Not a withdraw_validator_commission type")
	}
	se = shared.SubsetEvent{
		Type:   []string{"withdraw_delegator_reward"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: wdr.DelegatorAddress.String()}},
			"validator": {{ID: wdr.ValidatorAddress.String()}},
		},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wdr.DelegatorAddress.String()},
		}},
	}

	var withdrawAddr string
	rewards := []shared.TransactionAmount{}
	for _, ev := range logf.Events {
		if ev.Type != "transfer" {
			continue
		}

		for _, attr := range ev.Attributes {
			if len(attr.Recipient) > 0 {
				withdrawAddr = attr.Recipient[0]
			}

			for _, amount := range attr.Amount {
				attrAmt := shared.TransactionAmount{Numeric: &big.Int{}}
				sliced := getCurrency(amount)
				var (
					c       *big.Int
					exp     int32
					coinErr error
				)
				if len(sliced) == 3 {
					attrAmt.Currency = sliced[2]
					c, exp, coinErr = getCoin(sliced[1])
				} else {
					c, exp, coinErr = getCoin(amount)
				}
				if coinErr != nil {
					return se, fmt.Errorf("[COSMOS-API] Error parsing amount '%s': %s ", amount, coinErr)
				}
				attrAmt.Text = amount
				attrAmt.Numeric.Set(c)
				attrAmt.Exp = exp
				if attrAmt.Numeric.Cmp(&zero) != 0 {
					rewards = append(rewards, attrAmt)
				}
			}
		}
	}

	if len(rewards) == 0 {
		return se, nil
	}
	se.Transfers = map[string][]shared.EventTransfer{
		"reward": []shared.EventTransfer{{
			Amounts: rewards,
			Account: shared.Account{ID: withdrawAddr},
		}},
	}

	return se, nil
}

func mapDistributionFundCommunityPoolToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {

	fcp, ok := msg.(distributiontypes.MsgFundCommunityPool)
	if !ok {
		return se, errors.New("Not a withdraw_validator_commission type")
	}

	evt, err := distributionProduceEvTx(fcp.Depositor, fcp.Amount)
	return shared.SubsetEvent{
		Type:   []string{"fund_community_pool"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"depositor": {{ID: fcp.Depositor.String()}},
		},
		Sender: []shared.EventTransfer{evt},
	}, err

}

func distributionProduceEvTx(account sdk.AccAddress, coins sdk.Coins) (evt shared.EventTransfer, err error) {

	evt = shared.EventTransfer{
		Account: shared.Account{ID: account.String()},
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
