package mapper

import (
	"fmt"
	"math/big"

	"github.com/figment-networks/cosmos-worker/api/types"
	"github.com/figment-networks/cosmos-worker/api/util"
	shared "github.com/figment-networks/indexer-manager/structs"
)

const (
	TransferTypeSend   = "send"
	TransferTypeReward = "reward"
)

func produceTransfers(se *shared.SubsetEvent, transferType string, logf types.LogFormat) (err error) {
	var evts []shared.EventTransfer
	m := make(map[string][]shared.TransactionAmount)
	for _, ev := range logf.Events {
		if ev.Type != "transfer" {
			continue
		}

		var latestRecipient string
		for _, attr := range ev.Attributes {
			if len(attr.Recipient) > 0 {
				latestRecipient = attr.Recipient[0]
			}

			for _, amount := range attr.Amount {
				attrAmt := shared.TransactionAmount{Numeric: &big.Int{}}
				sliced := util.GetCurrency(amount)
				var (
					c       *big.Int
					exp     int32
					coinErr error
				)
				if len(sliced) == 3 {
					attrAmt.Currency = sliced[2]
					c, exp, coinErr = util.GetCoin(sliced[1])
				} else {
					c, exp, coinErr = util.GetCoin(amount)
				}
				if coinErr != nil {
					return fmt.Errorf("[COSMOS-API] Error parsing amount '%s': %s ", amount, coinErr)
				}

				attrAmt.Text = amount
				attrAmt.Exp = exp
				attrAmt.Numeric.Set(c)

				m[latestRecipient] = append(m[latestRecipient], attrAmt)

			}
		}
	}

	for addr, amts := range m {
		evts = append(evts, shared.EventTransfer{
			Amounts: amts,
			Account: shared.Account{ID: addr},
		})
	}

	if len(evts) <= 0 {
		return
	}

	if se.Transfers[transferType] == nil {
		se.Transfers = make(map[string][]shared.EventTransfer)
	}
	se.Transfers[transferType] = evts

	return
}
