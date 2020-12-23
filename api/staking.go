package api

import (
	"errors"
	"fmt"
	"math/big"

	shared "github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking"
)

const unbondedTokensPoolAddr = "cosmos1tygms3xhhs3yv487phx3dw4a95jn7t7lpm470r"

func mapStakingUndelegateToSub(msg sdk.Msg, logf LogFormat) (se shared.SubsetEvent, err error) {
	u, ok := msg.(staking.MsgUndelegate)
	if !ok {
		return se, errors.New("Not a begin_unbonding type")
	}

	se = shared.SubsetEvent{
		Type:   []string{"begin_unbonding"},
		Module: "staking",
		Node: map[string][]shared.Account{
			"delegator": {{ID: u.DelegatorAddress.String()}},
			"validator": {{ID: u.ValidatorAddress.String()}},
		},
		Amount: map[string]shared.TransactionAmount{
			"undelegate": {
				Currency: u.Amount.Denom,
				Numeric:  u.Amount.Amount.BigInt(),
				Text:     u.Amount.String(),
			},
		},
	}

	rewards := []shared.TransactionAmount{}
	for _, ev := range logf.Events {
		if ev.Type != "transfer" {
			continue
		}

		var latestRecipient string
		for _, attr := range ev.Attributes {
			if len(attr.Recipient) > 0 {
				latestRecipient = attr.Recipient[0]
			}
			if latestRecipient == unbondedTokensPoolAddr {
				continue
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
		"reward": []shared.EventTransfer{{Amounts: rewards}},
	}

	return se, nil
}

func mapStakingDelegateToSub(msg sdk.Msg, logf LogFormat) (se shared.SubsetEvent, err error) {
	d, ok := msg.(staking.MsgDelegate)
	if !ok {
		return se, errors.New("Not a delegate type")
	}
	se = shared.SubsetEvent{
		Type:   []string{"delegate"},
		Module: "staking",
		Node: map[string][]shared.Account{
			"delegator": {{ID: d.DelegatorAddress.String()}},
			"validator": {{ID: d.ValidatorAddress.String()}},
		},
		Amount: map[string]shared.TransactionAmount{
			"delegate": {
				Currency: d.Amount.Denom,
				Numeric:  d.Amount.Amount.BigInt(),
				Text:     d.Amount.String(),
			},
		},
	}

	rewards := []shared.TransactionAmount{}
	for _, ev := range logf.Events {
		if ev.Type != "transfer" {
			continue
		}

		for _, attr := range ev.Attributes {
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
				attrAmt.Exp = exp
				attrAmt.Numeric.Set(c)
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
		"reward": []shared.EventTransfer{{Amounts: rewards}},
	}
	return se, nil
}

func mapStakingBeginRedelegateToSub(msg sdk.Msg, logf LogFormat) (se shared.SubsetEvent, err error) {
	br, ok := msg.(staking.MsgBeginRedelegate)
	if !ok {
		return se, errors.New("Not a begin_redelegate type")
	}

	se = shared.SubsetEvent{
		Type:   []string{"begin_redelegate"},
		Module: "staking",
		Node: map[string][]shared.Account{
			"delegator":             {{ID: br.DelegatorAddress.String()}},
			"validator_destination": {{ID: br.ValidatorDstAddress.String()}},
			"validator_source":      {{ID: br.ValidatorDstAddress.String()}},
		},
		Amount: map[string]shared.TransactionAmount{
			"delegate": {
				Currency: br.Amount.Denom,
				Numeric:  br.Amount.Amount.BigInt(),
				Text:     br.Amount.String(),
			},
		},
	}

	rewards := []shared.TransactionAmount{}
	for _, ev := range logf.Events {
		if ev.Type != "transfer" {
			continue
		}

		for _, attr := range ev.Attributes {
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
				attrAmt.Exp = exp
				attrAmt.Numeric.Set(c)
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
		"reward": []shared.EventTransfer{{Amounts: rewards}},
	}
	return se, nil
}

func mapStakingCreateValidatorToSub(msg sdk.Msg) (se shared.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgCreateValidator)
	if !ok {
		return se, errors.New("Not a create_validator type")
	}
	return shared.SubsetEvent{
		Type:   []string{"create_validator"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: ev.DelegatorAddress.String()}},
			"validator": {
				{
					ID: ev.ValidatorAddress.String(),
					Details: &shared.AccountDetails{
						Name:        ev.Description.Moniker,
						Description: ev.Description.Details,
						Contact:     ev.Description.SecurityContact,
						Website:     ev.Description.Website,
					},
				},
			},
		},
		Amount: map[string]shared.TransactionAmount{
			"self_delegation": {
				Currency: ev.Value.Denom,
				Numeric:  ev.Value.Amount.BigInt(),
				Text:     ev.Value.String(),
			},
			"self_delegation_min": {
				Text:    ev.MinSelfDelegation.String(),
				Numeric: ev.MinSelfDelegation.BigInt(),
			},
			"commission_rate": {
				Text:    ev.Commission.Rate.String(),
				Numeric: ev.Commission.Rate.Int,
			},
			"commission_max_rate": {
				Text:    ev.Commission.MaxRate.String(),
				Numeric: ev.Commission.MaxRate.Int,
			},
			"commission_max_change_rate": {
				Text:    ev.Commission.MaxChangeRate.String(),
				Numeric: ev.Commission.MaxChangeRate.Int,
			}},
	}, err
}

func mapStakingEditValidatorToSub(msg sdk.Msg) (se shared.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgEditValidator)
	if !ok {
		return se, errors.New("Not a edit_validator type")
	}
	sev := shared.SubsetEvent{
		Type:   []string{"edit_validator"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"validator": {
				{
					ID: ev.ValidatorAddress.String(),
					Details: &shared.AccountDetails{
						Name:        ev.Description.Moniker,
						Description: ev.Description.Details,
						Contact:     ev.Description.SecurityContact,
						Website:     ev.Description.Website,
					},
				},
			},
		},
	}

	if ev.MinSelfDelegation != nil || ev.CommissionRate != nil {
		sev.Amount = map[string]shared.TransactionAmount{}
		if ev.MinSelfDelegation != nil {
			sev.Amount["self_delegation_min"] = shared.TransactionAmount{
				Text:    ev.MinSelfDelegation.String(),
				Numeric: ev.MinSelfDelegation.BigInt(),
			}
		}

		if ev.CommissionRate != nil {
			sev.Amount["commission_rate"] = shared.TransactionAmount{
				Text:    ev.CommissionRate.String(),
				Numeric: ev.CommissionRate.Int,
			}
		}
	}
	return sev, err
}
