package mapper

import (
	"fmt"
	"math/big"
	"strings"

	// "github.com/figment-networks/cosmos-worker/api/types"
	// "github.com/figment-networks/cosmos-worker/api/util"
	"github.com/figment-networks/cosmos-worker/api/util"
	"github.com/figment-networks/indexer-manager/structs"

	"github.com/cosmos/cosmos-sdk/types"
)

func produceTransfers(se *structs.SubsetEvent, transferType, skipAddr string, lg types.ABCIMessageLog) (err error) {
	var evts []structs.EventTransfer

	m := make(map[string][]structs.TransactionAmount)
	for _, ev := range lg.GetEvents() {

		if ev.GetType() != "transfer" {
			continue
		}

		var latestRecipient string
		for _, attr := range ev.GetAttributes() {
			if attr.Key == "recipient" {
				latestRecipient = attr.Value
			}

			if latestRecipient == skipAddr {
				continue
			}

			if attr.Key == "amount" {
				amounts := strings.Split(attr.Value, ",")
				for _, amt := range amounts {
					attrAmt := structs.TransactionAmount{Numeric: &big.Int{}}

					sliced := util.GetCurrency(amt)
					var (
						c       *big.Int
						exp     int32
						coinErr error
					)
					if len(sliced) == 3 {
						attrAmt.Currency = sliced[2]
						c, exp, coinErr = util.GetCoin(sliced[1])
					} else {
						c, exp, coinErr = util.GetCoin(amt)
					}
					if coinErr != nil {
						return fmt.Errorf("[COSMOS-API] Error parsing amount '%s': %s ", amt, coinErr)
					}

					attrAmt.Text = amt
					attrAmt.Exp = exp
					attrAmt.Numeric.Set(c)

					m[latestRecipient] = append(m[latestRecipient], attrAmt)
				}
			}
		}
	}

	for addr, amts := range m {
		evts = append(evts, structs.EventTransfer{
			Amounts: amts,
			Account: structs.Account{ID: addr},
		})
	}

	if len(evts) <= 0 {
		return
	}

	if se.Transfers[transferType] == nil {
		se.Transfers = make(map[string][]structs.EventTransfer)
	}
	se.Transfers[transferType] = evts
	return
}
