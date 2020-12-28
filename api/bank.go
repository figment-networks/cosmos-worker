package api

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
)

func mapBankMultisendToSub(msg sdk.Msg, logf LogFormat) (se shared.SubsetEvent, err error) {

	multisend, ok := msg.(bank.MsgMultiSend)
	if !ok {
		return se, errors.New("Not a multisend type")
	}

	se = shared.SubsetEvent{
		Type:   []string{"multisend"},
		Module: "bank",
	}
	for _, i := range multisend.Inputs {
		evt, err := bankProduceEvTx(i.Address, i.Coins)
		if err != nil {
			continue
		}
		se.Sender = append(se.Sender, evt)
	}

	for _, o := range multisend.Outputs {
		evt, err := bankProduceEvTx(o.Address, o.Coins)
		if err != nil {
			continue
		}
		se.Recipient = append(se.Recipient, evt)
	}

	err = produceTransfers(&se, "send", logf)
	return se, err
}

func mapBankSendToSub(msg sdk.Msg, logf LogFormat) (se shared.SubsetEvent, err error) {
	send, ok := msg.(bank.MsgSend)
	if !ok {
		return se, errors.New("Not a send type")
	}

	se = shared.SubsetEvent{
		Type:   []string{"send"},
		Module: "bank",
	}

	evt, _ := bankProduceEvTx(send.FromAddress, send.Amount)
	se.Sender = append(se.Sender, evt)

	evt, _ = bankProduceEvTx(send.ToAddress, send.Amount)
	se.Recipient = append(se.Recipient, evt)

	err = produceTransfers(&se, "send", logf)
	return se, err
}

func bankProduceEvTx(account sdk.AccAddress, coins sdk.Coins) (evt shared.EventTransfer, err error) {

	evt = shared.EventTransfer{
		Account: shared.Account{ID: account.String()},
	}
	if len(coins) > 0 {
		evt.Amounts = []shared.TransactionAmount{}
		for _, coin := range coins {
			evt.Amounts = append(evt.Amounts, shared.TransactionAmount{
				Currency: coin.Denom,
				Numeric:  coin.Amount.BigInt(),
				Text:     coin.Amount.String(),
			})
		}
	}

	return evt, nil
}
