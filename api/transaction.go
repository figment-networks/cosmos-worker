package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/figment-networks/cosmos-worker/api/mapper"
	"github.com/figment-networks/indexer-manager/structs"

	codec_types "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"go.uber.org/zap"
)

// TxLogError Error message
type TxLogError struct {
	Codespace string  `json:"codespace"`
	Code      float64 `json:"code"`
	Message   string  `json:"message"`
}

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-\\s]+)([^0-9\\s]+)$")

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, r structs.HeightHash, block structs.Block, perPage uint64) (txs []structs.Transaction, err error) {
	pag := &query.PageRequest{
		CountTotal: true,
		Limit:      perPage,
	}

	var page = uint64(1)
	for {
		pag.Offset = (perPage * page) - perPage
		now := time.Now()

		if err = c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
		nctx, cancel := context.WithTimeout(ctx, time.Second*10)

		grpcRes, err := c.txServiceClient.GetTxsEvent(nctx, &tx.GetTxsEventRequest{
			//Events: []string{"message.action=submit_evidence"},
			Events:     []string{"tx.height=" + strconv.FormatUint(r.Height, 10)},
			Pagination: pag,
		})
		cancel()

		c.logger.Debug("[COSMOS-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
		if err != nil {
			return nil, err
		}
		rawRequestDuration.WithLabels("/tx_search", "200").Observe(time.Since(now).Seconds())
		numberOfItemsTransactions.Observe(float64(len(grpcRes.Txs)))

		for i, trans := range grpcRes.Txs {
			resp := grpcRes.TxResponses[i]
			tx, err := rawToTransaction(ctx, trans, resp, c.logger)
			if err != nil {
				return nil, err
			}
			tx.BlockHash = block.Hash
			tx.ChainID = block.ChainID
			tx.Time = block.Time
			txs = append(txs, tx)
		}

		if grpcRes.Pagination.GetTotal() <= uint64(len(txs)) {
			break
		}

		page++

	}

	c.logger.Debug("[COSMOS-API] Sending requests ", zap.Int("number", len(txs)))
	return txs, nil
}

// transform raw data from cosmos into transaction format with augmentation from blocks
func rawToTransaction(ctx context.Context, in *tx.Tx, resp *types.TxResponse, logger *zap.Logger) (trans structs.Transaction, err error) {

	trans = structs.Transaction{
		Height:    uint64(resp.Height),
		Hash:      resp.TxHash,
		GasWanted: uint64(resp.GasWanted),
		GasUsed:   uint64(resp.GasUsed),
	}

	if resp.RawLog != "" {
		trans.RawLog = []byte(resp.RawLog)
	} else {
		trans.RawLog = []byte(resp.Logs.String())
	}

	trans.Raw, err = in.Marshal()
	if err != nil {
		return trans, errors.New("Error marshaling tx to raw")
	}

	if in.Body != nil {
		trans.Memo = in.Body.Memo

		for index, m := range in.Body.Messages {
			tev := structs.TransactionEvent{
				ID: strconv.Itoa(index),
			}
			lg := findLog(resp.Logs, index)

			// tPath is "/cosmos.bank.v1beta1.MsgSend" or "/ibc.core.client.v1.MsgCreateClient"
			tPath := strings.Split(m.TypeUrl, ".")

			var err error
			var msgType string
			if len(tPath) == 5 && tPath[0] == "/ibc" {
				msgType = tPath[4]
				err = addIBCSubEvent(tPath[2], msgType, &tev, m, lg, logger)
			} else if len(tPath) == 4 && tPath[0] == "/cosmos" {
				msgType = tPath[3]
				err = addSubEvent(tPath[1], msgType, &tev, m, lg, logger)
			} else {
				return trans, fmt.Errorf("TypeURL is in wrong format: %v", m.TypeUrl)
			}

			if err != nil {
				logger.Error("[COSMOS-API] Problem decoding transaction ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
			}

			trans.Events = append(trans.Events, tev)
		}
	}

	if in.AuthInfo != nil {
		for _, coin := range in.AuthInfo.Fee.Amount {
			trans.Fee = append(trans.Fee, structs.TransactionAmount{
				Text:     coin.Amount.String(),
				Numeric:  coin.Amount.BigInt(),
				Currency: coin.Denom,
			})
		}
	}

	if resp.Code > 0 {
		trans.Events = append(trans.Events, structs.TransactionEvent{
			Kind: "error",
			Sub: []structs.SubsetEvent{{
				Type:   []string{"error"},
				Module: resp.Codespace,
				Error:  &structs.SubsetEventError{Message: resp.Info},
			}},
		})
	}

	return trans, nil
}

func findLog(logs types.ABCIMessageLogs, index int) types.ABCIMessageLog {
	if len(logs) <= index {
		return types.ABCIMessageLog{}
	}
	if lg := logs[index]; lg.GetMsgIndex() == uint32(index) {
		return lg
	}
	for _, lg := range logs {
		if lg.GetMsgIndex() == uint32(index) {
			return lg
		}
	}
	return types.ABCIMessageLog{}
}

func addSubEvent(msgRoute, msgType string, tev *structs.TransactionEvent, m *codec_types.Any, lg types.ABCIMessageLog, logger *zap.Logger) (err error) {
	var ev structs.SubsetEvent
	switch msgRoute {
	case "bank":
		switch msgType {
		case "MsgSend":
			tev.Kind = "send"
			ev, err = mapper.BankSendToSub(m.Value, lg)
		case "MsgMultiSend":
			tev.Kind = "multisend"
			ev, err = mapper.BankMultisendToSub(m.Value, lg)
		default:
			logger.Error("[COSMOS-API] Unknown bank message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}
	case "crisis":
		switch msgType {
		case "MsgVerifyInvariant":
			tev.Kind = "verify_invariant"
			ev, err = mapper.CrisisVerifyInvariantToSub(m.Value)
		default:
			logger.Error("[COSMOS-API] Unknown crisis message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}
	case "distribution":
		switch msgType {
		case "MsgWithdrawValidatorCommission":
			tev.Kind = "withdraw_validator_commission"
			ev, err = mapper.DistributionWithdrawValidatorCommissionToSub(m.Value, lg)
		case "MsgSetWithdrawAddress":
			tev.Kind = "set_withdraw_address"
			ev, err = mapper.DistributionSetWithdrawAddressToSub(m.Value)
		case "MsgWithdrawDelegatorReward":
			tev.Kind = "withdraw_delegator_reward"
			ev, err = mapper.DistributionWithdrawDelegatorRewardToSub(m.Value, lg)
		case "MsgFundCommunityPool":
			tev.Kind = "fund_community_pool"
			ev, err = mapper.DistributionFundCommunityPoolToSub(m.Value)
		default:
			logger.Error("[COSMOS-API] Unknown distribution message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}
	case "evidence":
		switch msgType {
		case "MsgSubmitEvidence":
			tev.Kind = "submit_evidence"
			ev, err = mapper.EvidenceSubmitEvidenceToSub(m.Value)
		default:
			logger.Error("[COSMOS-API] Unknown evidence message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}
	case "gov":
		switch msgType {
		case "MsgDeposit":
			tev.Kind = "deposit"
			ev, err = mapper.GovDepositToSub(m.Value, lg)
		case "MsgVote":
			tev.Kind = "vote"
			ev, err = mapper.GovVoteToSub(m.Value)
		case "MsgSubmitProposal":
			tev.Kind = "submit_proposal"
			ev, err = mapper.GovSubmitProposalToSub(m.Value, lg)
		default:
			logger.Error("[COSMOS-API] Unknown got message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}
	case "slashing":
		switch msgType {
		case "MsgUnjail":
			tev.Kind = "unjail"
			ev, err = mapper.SlashingUnjailToSub(m.Value)
		default:
			logger.Error("[COSMOS-API] Unknown slashing message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}
	case "staking":
		switch msgType {
		case "MsgUndelegate":
			tev.Kind = "begin_unbonding"
			ev, err = mapper.StakingUndelegateToSub(m.Value, lg)
		case "MsgEditValidator":
			tev.Kind = "edit_validator"
			ev, err = mapper.StakingEditValidatorToSub(m.Value)
		case "MsgCreateValidator":
			tev.Kind = "create_validator"
			ev, err = mapper.StakingCreateValidatorToSub(m.Value)
		case "MsgDelegate":
			tev.Kind = "delegate"
			ev, err = mapper.StakingDelegateToSub(m.Value, lg)
		case "MsgBeginRedelegate":
			tev.Kind = "begin_redelegate"
			ev, err = mapper.StakingBeginRedelegateToSub(m.Value, lg)
		default:
			logger.Error("[COSMOS-API] Unknown staking message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}
	default:
		logger.Error("[COSMOS-API] Unknown message Route ", zap.Error(err), zap.String("route", msgType), zap.String("type", m.TypeUrl))
	}

	if len(ev.Type) > 0 {
		tev.Sub = append(tev.Sub, ev)
	}
	return err
}

func addIBCSubEvent(msgRoute, msgType string, tev *structs.TransactionEvent, m *codec_types.Any, lg types.ABCIMessageLog, logger *zap.Logger) (err error) {
	var ev structs.SubsetEvent

	switch msgRoute {
	case "client":
		switch msgType {
		case "MsgCreateClient":
			tev.Kind = "create_client"
			ev, err = mapper.IBCCreateClientToSub(m.Value)
		case "MsgUpdateClient":
			tev.Kind = "update_client"
			ev, err = mapper.IBCCreateClientToSub(m.Value)
		case "MsgUpgradeClient":
			tev.Kind = "upgrade_client"
			ev, err = mapper.IBCCreateClientToSub(m.Value)
		case "MsgSubmitMisbehaviour":
			tev.Kind = "submit_misbehaviour"
			ev, err = mapper.IBCCreateClientToSub(m.Value)
		}
	case "connection":
		switch msgType {
		case "MsgConnectionOpenInit":
			tev.Kind = "connection_open_init"
			ev, err = mapper.IBCConnectionOpenInitToSub(m.Value)
		case "MsgConnectionOpenConfirm":
			tev.Kind = "connection_open_confirm"
			ev, err = mapper.IBCConnectionOpenConfirmToSub(m.Value)
		case "MsgConnectionOpenAck":
			tev.Kind = "connection_open_ack"
			ev, err = mapper.IBCConnectionOpenAckToSub(m.Value)
		case "MsgConnectionOpenTry":
			tev.Kind = "connection_open_try"
			ev, err = mapper.IBCConnectionOpenTryToSub(m.Value)
		default:
			logger.Error("[COSMOS-API] Unknown got message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}

	case "channel":
		switch msgType {
		case "MsgChannelOpenInit":
			tev.Kind = "channel_open_init"
			ev, err = mapper.IBCChannelOpenInitToSub(m.Value)
		default:
			logger.Error("[COSMOS-API] Unknown got message Type ", zap.Error(err), zap.String("type", msgType), zap.String("route", m.TypeUrl))
		}

	default:
		logger.Error("[COSMOS-API] Unknown message Route ", zap.Error(err), zap.String("route", msgType), zap.String("type", m.TypeUrl))
	}

	if len(ev.Type) > 0 {
		tev.Sub = append(tev.Sub, ev)
	}
	return err
}

/*
// transform raw data from cosmos into transaction format with augmentation from blocks
func rawToTransaction(ctx context.Context, c *Client, in []TxResponse, blocks map[uint64]structs.Block, out chan cStruct.OutResp, logger *zap.Logger, cdc *codec.Codec) error {
	defer logger.Sync()
	for _, txRaw := range in {
		timer := metrics.NewTimer(transactionConversionDuration)
		tx := &auth.StdTx{}
		lf := []LogFormat{}
		txErrs := []TxLogError{}

		if err := json.Unmarshal([]byte(txRaw.TxResult.Log), &lf); err != nil {
			// (lukanus): Try to fallback to known error format
			tle := TxLogError{}
			if errin := json.Unmarshal([]byte(txRaw.TxResult.Log), &tle); errin != nil {
				logger.Error("[COSMOS-API] Problem decoding raw transaction (json)", zap.Error(err), zap.String("content_log", txRaw.TxResult.Log), zap.Any("content", txRaw))
			}
			if tle.Message != "" {
				txErrs = append(txErrs, tle)
			}
		}

		for _, logf := range lf {
			tle := TxLogError{}
			if errin := json.Unmarshal([]byte(logf.Log), &tle); errin == nil && tle.Message != "" {
				txErrs = append(txErrs, tle)
			}
		}

		txReader := strings.NewReader(txRaw.TxData)
		base64Dec := base64.NewDecoder(base64.StdEncoding, txReader)
		_, err := cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
		if err != nil {
			logger.Error("[COSMOS-API] Problem decoding raw transaction (cdc)", zap.Error(err), zap.Any("height", txRaw.Height), zap.Any("raw_tx", txRaw))
		}
		hInt, err := strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			logger.Error("[COSMOS-API] Problem parsing height", zap.Error(err))
		}

		outTX := cStruct.OutResp{Type: "Transaction"}
		block := blocks[hInt]
		trans := structs.Transaction{
			Hash:      txRaw.Hash,
			Memo:      tx.GetMemo(),
			Time:      block.Time,
			ChainID:   block.ChainID,
			BlockHash: block.Hash,
			RawLog:    []byte(txRaw.TxResult.Log),
		}

		for _, coin := range tx.Fee.Amount {
			trans.Fee = append(trans.Fee, structs.TransactionAmount{
				Text:     coin.Amount.String(),
				Numeric:  coin.Amount.BigInt(),
				Currency: coin.Denom,
			})
		}

		trans.Height, err = strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasWanted, err = strconv.ParseUint(txRaw.TxResult.GasWanted, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasUsed, err = strconv.ParseUint(txRaw.TxResult.GasUsed, 10, 64)
		if err != nil {
			outTX.Error = err
		}

		txReader.Seek(0, 0)
		trans.Raw = make([]byte, txReader.Len())
		txReader.Read(trans.Raw)

		presentIndexes := map[string]bool{}

		for index, msg := range tx.Msgs {
			tev := structs.TransactionEvent{
				ID: strconv.Itoa(index),
			}

			var ev structs.SubsetEvent
			var err error

			switch msg.Route() {
			case "bank":
				switch msg.Type() {
				case "multisend":
					ev, err = mapper.BankMultisendToSub(msg)
				case "send":
					ev, err = mapper.BankSendToSub(msg)
				default:
					c.logger.Error("[COSMOS-API] Unknown bank message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "crisis":
				switch msg.Type() {
				case "verify_invariant":
					ev, err = mapper.CrisisVerifyInvariantToSub(msg)
				default:
					c.logger.Error("[COSMOS-API] Unknown crisis message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "distribution":
				switch msg.Type() {
				case "withdraw_validator_commission":
					ev, err = mapper.DistributionWithdrawValidatorCommissionToSub(msg)
				case "set_withdraw_address":
					ev, err = mapper.DistributionSetWithdrawAddressToSub(msg)
				case "withdraw_delegator_reward":
					ev, err = mapper.DistributionWithdrawDelegatorRewardToSub(msg)
				case "fund_community_pool":
					ev, err = mapper.DistributionFundCommunityPoolToSub(msg)
				default:
					c.logger.Error("[COSMOS-API] Unknown distribution message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "evidence":
				switch msg.Type() {
				case "submit_evidence":
					ev, err = mapper.EvidenceSubmitEvidenceToSub(msg)
				default:
					c.logger.Error("[COSMOS-API] Unknown evidence message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "gov":
				switch msg.Type() {
				case "deposit":
					ev, err = mapper.GovDepositToSub(msg)
				case "vote":
					ev, err = mapper.GovVoteToSub(msg)
				case "submit_proposal":
					ev, err = mapper.GovSubmitProposalToSub(msg)
				default:
					c.logger.Error("[COSMOS-API] Unknown got message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "slashing":
				switch msg.Type() {
				case "unjail":
					ev, err = mapper.SlashingUnjailToSub(msg)
				default:
					c.logger.Error("[COSMOS-API] Unknown slashing message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "staking":
				switch msg.Type() {
				case "begin_unbonding":
					ev, err = mapper.StakingUndelegateToSub(msg)
				case "edit_validator":
					ev, err = mapper.StakingEditValidatorToSub(msg)
				case "create_validator":
					ev, err = mapper.StakingCreateValidatorToSub(msg)
				case "delegate":
					ev, err = mapper.StakingDelegateToSub(msg)
				case "begin_redelegate":
					ev, err = mapper.StakingBeginRedelegateToSub(msg)
				default:
					c.logger.Error("[COSMOS-API] Unknown staking message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			default:
				c.logger.Error("[COSMOS-API] Unknown message Route ", zap.Error(err), zap.String("route", msg.Route()), zap.String("type", msg.Type()))
			}

			if len(ev.Type) > 0 {
				tev.Kind = msg.Type()
				tev.Sub = append(tev.Sub, ev)
			}

			if err != nil {
				c.logger.Error("[COSMOS-API] Problem decoding transaction ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
			}

			presentIndexes[tev.ID] = true
			trans.Events = append(trans.Events, tev)
		}

		for _, logf := range lf {
			msgIndex := strconv.FormatFloat(logf.MsgIndex, 'f', -1, 64)
			_, ok := presentIndexes[msgIndex]
			if ok {
				continue
			}

			tev := structs.TransactionEvent{
				ID: msgIndex,
			}
			for _, ev := range logf.Events {
				sub := structs.SubsetEvent{
					Type: []string{ev.Type},
				}
				for atk, attr := range ev.Attributes {
					sub.Module = attr.Module
					sub.Action = attr.Action

					if len(attr.Sender) > 0 {
						for _, senderID := range attr.Sender {
							sub.Sender = append(sub.Sender, structs.EventTransfer{Account: structs.Account{ID: senderID}})
						}
					}
					if len(attr.Recipient) > 0 {
						for _, recipientID := range attr.Recipient {
							sub.Recipient = append(sub.Recipient, structs.EventTransfer{Account: structs.Account{ID: recipientID}})
						}
					}
					if attr.CompletionTime != "" {
						cTime, _ := time.Parse(time.RFC3339Nano, attr.CompletionTime)
						sub.Completion = &cTime
					}
					if len(attr.Validator) > 0 {
						if sub.Node == nil {
							sub.Node = make(map[string][]structs.Account)
						}
						for k, v := range attr.Validator {
							w, ok := sub.Node[k]
							if !ok {
								w = []structs.Account{}
							}

							for _, validatorID := range v {
								w = append(w, structs.Account{ID: validatorID})
							}
							sub.Node[k] = w
						}
					}

					for index, amount := range attr.Amount {
						sliced := getCurrency(amount)

						am := structs.TransactionAmount{
							Text: amount,
						}

						var (
							c       *big.Int
							exp     int32
							coinErr error
						)

						if len(sliced) == 3 {
							am.Currency = sliced[2]
							c, exp, coinErr = getCoin(sliced[1])
						} else {
							c, exp, coinErr = getCoin(amount)
						}

						if coinErr != nil {
							am.Numeric.Set(c)
							am.Exp = exp
						}

						if sub.Amount == nil {
							sub.Amount = make(map[string]structs.TransactionAmount)
						}
						sub.Amount[strconv.Itoa(index)] = am
					}
					ev.Attributes[atk] = nil
				}
				tev.Sub = append(tev.Sub, sub)
			}
			logf.Events = nil
			trans.Events = append(trans.Events, tev)
		}

		for _, txErr := range txErrs {
			if txErr.Message != "" {
				trans.Events = append(trans.Events, structs.TransactionEvent{
					Kind: "error",
					Sub: []structs.SubsetEvent{{
						Type:   []string{"error"},
						Module: txErr.Codespace,
						Error:  &structs.SubsetEventError{Message: txErr.Message},
					}},
				})
			}
		}

		outTX.Payload = trans
		out <- outTX
		timer.ObserveDuration()

		// GC Help
		lf = nil

	}

	return nil
}
*/
/*
func getCurrency(in string) []string {
	return curencyRegex.FindStringSubmatch(in)
}

func getCoin(s string) (number *big.Int, exp int32, err error) {
	s = strings.Replace(s, ",", ".", -1)
	strs := strings.Split(s, `.`)
	if len(strs) == 1 {
		i := &big.Int{}
		i.SetString(strs[0], 10)
		return i, 0, nil
	}
	if len(strs) == 2 {
		i := &big.Int{}
		i.SetString(strs[0]+strs[1], 10)
		return i, int32(len(strs[1])), nil
	}

	return number, 0, errors.New("Impossible to parse ")
}
*/
/*
// GetFromRaw returns raw data for plugin use;
func (c *Client) GetFromRaw(logger *zap.Logger, txReader io.Reader) []map[string]interface{} {/	tx := &auth.StdTx{}
	base64Dec := base64.NewDecoder(base64.StdEncoding, txReader)
	_, err := c.cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
	if err != nil {
		logger.Error("[COSMOS-API] Problem decoding raw transaction (cdc) ", zap.Error(err))
	}
	slice := []map[string]interface{}{}
	for _, coin := range tx.Fee.Amount {
		slice = append(slice, map[string]interface{}{
			"text":     coin.Amount.String(),
			"numeric":  coin.Amount.BigInt(),
			"currency": coin.Denom,
		})
	}
	return slice
}
*/
