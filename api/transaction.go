package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/figment-networks/cosmos-worker/api/mapper"
	"github.com/figment-networks/cosmos-worker/api/types"
	"github.com/figment-networks/cosmos-worker/api/util"
	"github.com/figment-networks/indexer-manager/structs"
	cStruct "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/indexing-engine/metrics"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

// TxLogError Error message
type TxLogError struct {
	Codespace string  `json:"codespace"`
	Code      float64 `json:"code"`
	Message   string  `json:"message"`
}

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, r structs.HeightHash, block structs.Block /*out chan cStruct.OutResp, page, perPage int */) (txs []structs.Transaction, err error) {

	grpcRes, err := c.txServiceClient.GetTxsEvent(ctx, &tx.GetTxsEventRequest{
		Events: []string{"tx.height=" + strconv.FormatUint(r.Height, 10)},
		/*Pagination: &query.PageRequest{
			CountTotal: true,
			Offset:     0,
			Limit:      30,
		},*/
	})
	if err != nil {
		return nil, err
	}

	for i, trans := range grpcRes.Txs {
		resp := grpcRes.TxResponses[i]
		tx, err := rawToTransaction(ctx, trans, resp, c.logger)
		if err != nil {
			return nil, err
		}
		tx.BlockHash = block.Hash
		tx.ChainID = block.ChainID

		txs = append(txs, tx)
	}

	return txs, nil
}

// transform raw data from cosmos into transaction format with augmentation from blocks
func rawToTransaction(ctx context.Context, in *tx.Tx, resp *types.TxResponse, logger *zap.Logger) (trans structs.Transaction, err error) {

	trans = structs.Transaction{
		Memo:      in.Body.Memo,
		Height:    uint64(resp.Height),
		Hash:      resp.TxHash,
		GasWanted: uint64(resp.GasWanted),
		GasUsed:   uint64(resp.GasUsed),
	}

	for index, m := range in.Body.Messages {
		// tPath is "/cosmos.bank.v1beta1.MsgSend"
		tPath := strings.Split(m.TypeUrl, ".")

		if len(tPath) != 4 {
			return trans, errors.New("TypeURL is in wrong format")
		}

		if tPath[0] != "/cosmos" {
			return trans, errors.New("TypeURL is not cosmos type")
		}

		tev := structs.TransactionEvent{
			ID: strconv.Itoa(index),
		}

		var ev structs.SubsetEvent
		var err error

		switch tPath[1] {
		case "bank":
			switch tPath[3] {
			case "MsgSend":
				ev, err = mapBankSendToSub(m.Value)
			case "MsgMultiSend":
				ev, err = mapBankMultisendToSub(m.Value)
			default:
				logger.Error("[COSMOS-API] Unknown bank message Type ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
			}
		case "crisis":
			switch tPath[3] {
			case "MsgVerifyInvariant":
				ev, err = mapCrisisVerifyInvariantToSub(m.Value)
			default:
				logger.Error("[COSMOS-API] Unknown crisis message Type ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
			}
		case "distribution":
			switch tPath[3] {
			case "MsgWithdrawValidatorCommission":
				ev, err = mapDistributionWithdrawValidatorCommissionToSub(m.Value)
			case "MsgSetWithdrawAddress":
				ev, err = mapDistributionSetWithdrawAddressToSub(m.Value)
			case "MsgWithdrawDelegatorReward":
				ev, err = mapDistributionWithdrawDelegatorRewardToSub(m.Value)
			case "MsgFundCommunityPool":
				ev, err = mapDistributionFundCommunityPoolToSub(m.Value)
			default:
				logger.Error("[COSMOS-API] Unknown distribution message Type ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
			}
		case "evidence":
			switch tPath[3] {
			case "MsgSubmitEvidence":
				ev, err = mapEvidenceSubmitEvidenceToSub(m.Value)
			default:
				logger.Error("[COSMOS-API] Unknown evidence message Type ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
			}
		case "gov":
			switch tPath[3] {
			case "MsgDeposit":
				ev, err = mapGovDepositToSub(m.Value)
			case "MsgVote":
				ev, err = mapGovVoteToSub(m.Value)
			case "MsgSubmitProposal":
				ev, err = mapGovSubmitProposalToSub(m.Value)
			default:
				logger.Error("[COSMOS-API] Unknown got message Type ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
			}
		case "slashing":
			switch tPath[3] {
			case "MsgUnjail":
				ev, err = mapSlashingUnjailToSub(m.Value)
			default:
				logger.Error("[COSMOS-API] Unknown slashing message Type ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
			}
		case "staking":
			switch tPath[3] {
			case "MsgUndelegate":
				ev, err = mapStakingUndelegateToSub(m.Value)
			case "MsgEditValidator":
				ev, err = mapStakingEditValidatorToSub(m.Value)
			case "MsgCreateValidator":
				ev, err = mapStakingCreateValidatorToSub(m.Value)
			case "MsgDelegate":
				ev, err = mapStakingDelegateToSub(m.Value)
			case "MsgBeginRedelegate":
				ev, err = mapStakingBeginRedelegateToSub(m.Value)
			default:
				logger.Error("[COSMOS-API] Unknown staking message Type ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
			}
		default:
			logger.Error("[COSMOS-API] Unknown message Route ", zap.Error(err), zap.String("route", tPath[3]), zap.String("type", m.TypeUrl))
		}

		if len(ev.Type) > 0 {
			tev.Kind = tPath[3]
			tev.Sub = append(tev.Sub, ev)
		}

		if err != nil {
			logger.Error("[COSMOS-API] Problem decoding transaction ", zap.Error(err), zap.String("type", tPath[3]), zap.String("route", m.TypeUrl))
		}

		//presentIndexes[tev.ID] = true
		trans.Events = append(trans.Events, tev)
	}

	/*
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


	*/
	return trans, nil
}

/*
	grpcRes, err := c.txServiceClient.GetTxsEvent(
		ctx,
		&tx.GetTxsEventRequest{
			Events: []string{"height=200"},
			//Events: []string{"message.height=200"},
			Pagination: &query.PageRequest{
				CountTotal: true,
				Offset:     page,
				Limit:      perPage,
			},
		},
	)*/

/*
// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, r structs.HeightRange, blocks map[uint64]structs.Block, out chan cStruct.OutResp, page, perPage int, fin chan string) {
	defer c.logger.Sync()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		fin <- err.Error()
		return
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	s := strings.Builder{}

	s.WriteString(`"`)

	if r.EndHeight > 0 && r.EndHeight != r.StartHeight {
		s.WriteString("tx.height>=")
		s.WriteString(strconv.FormatUint(r.StartHeight, 10))
		s.WriteString(" AND tx.height<=")
		s.WriteString(strconv.FormatUint(r.EndHeight, 10))
	} else {
		s.WriteString("tx.height=")
		s.WriteString(strconv.FormatUint(r.StartHeight, 10))
	}
	s.WriteString(`"`)

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	// (lukanus): do not block initial calls
	if r.EndHeight != 0 && r.StartHeight != 0 {
		err = c.rateLimiter.Wait(ctx)
		if err != nil {
			fin <- err.Error()
			return
		}
	}

	now := time.Now()
	resp, err := c.httpClient.Do(req)

	log.Debug("[COSMOS-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
	if err != nil {
		fin <- err.Error()
		return
	}

	if resp.StatusCode > 399 { // ERROR
		serverError, _ := ioutil.ReadAll(resp.Body)

		c.logger.Error("[COSMOS-API] error getting response from server", zap.Int("code", resp.StatusCode), zap.Any("response", string(serverError)))
		err := fmt.Errorf("error getting response from server %d %s", resp.StatusCode, string(serverError))
		fin <- err.Error()
		return
	}

	rawRequestDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &types.GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		c.logger.Error("[COSMOS-API] unable to decode result body", zap.Error(err))
		err := fmt.Errorf("unable to decode result body %w", err)
		fin <- err.Error()
		return
	}

	if result.Error.Message != "" {
		c.logger.Error("[COSMOS-API] Error getting search", zap.Any("result", result.Error.Message))
		err := fmt.Errorf("Error getting search: %s", result.Error.Message)
		fin <- err.Error()
		return
	}

	totalCount, err := strconv.ParseInt(result.Result.TotalCount, 10, 64)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery), zap.Any("request", r))
		fin <- err.Error()
		return
	}

	numberOfItemsTransactions.Observe(float64(totalCount))
	c.logger.Debug("[COSMOS-API] Converting requests ", zap.Int("number", len(result.Result.Txs)), zap.Int("blocks", len(blocks)))
	err = rawToTransaction(ctx, c, result.Result.Txs, blocks, out, c.logger, c.cdc)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting rawToTransaction", zap.Error(err))
		fin <- err.Error()
	}
	c.logger.Debug("[COSMOS-API] Converted all requests ")

	fin <- ""
	return
}

// transform raw data from cosmos into transaction format with augmentation from blocks
func rawToTransaction(ctx context.Context, c *Client, in []types.TxResponse, blocks map[uint64]structs.Block, out chan cStruct.OutResp, logger *zap.Logger, cdc *codec.Codec) error {
	defer logger.Sync()
	for _, txRaw := range in {
		timer := metrics.NewTimer(transactionConversionDuration)
		tx := &auth.StdTx{}
		lf := []types.LogFormat{}
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
			logAtIndex := findLog(lf, index)

			switch msg.Route() {
			case "bank":
				switch msg.Type() {
				case "multisend":
					ev, err = mapper.BankMultisendToSub(msg, logAtIndex)
				case "send":
					ev, err = mapper.BankSendToSub(msg, logAtIndex)
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
					ev, err = mapper.DistributionWithdrawValidatorCommissionToSub(msg, logAtIndex)
				case "set_withdraw_address":
					ev, err = mapper.DistributionSetWithdrawAddressToSub(msg)
				case "withdraw_delegator_reward":
					ev, err = mapper.DistributionWithdrawDelegatorRewardToSub(msg, logAtIndex)
				case "fund_community_pool":
					ev, err = mapper.DistributionFundCommunityPoolToSub(msg, logAtIndex)
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
					ev, err = mapper.GovDepositToSub(msg, logAtIndex)
				case "vote":
					ev, err = mapper.GovVoteToSub(msg)
				case "submit_proposal":
					ev, err = mapper.GovSubmitProposalToSub(msg, logAtIndex)
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
					ev, err = mapper.StakingUndelegateToSub(msg, logAtIndex)
				case "edit_validator":
					ev, err = mapper.StakingEditValidatorToSub(msg)
				case "create_validator":
					ev, err = mapper.StakingCreateValidatorToSub(msg)
				case "delegate":
					ev, err = mapper.StakingDelegateToSub(msg, logAtIndex)
				case "begin_redelegate":
					ev, err = mapper.StakingBeginRedelegateToSub(msg, logAtIndex)
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
						sliced := util.GetCurrency(amount)

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
							c, exp, coinErr = util.GetCoin(sliced[1])
						} else {
							c, exp, coinErr = util.GetCoin(amount)
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

// GetFromRaw returns raw data for plugin use;
func (c *Client) GetFromRaw(logger *zap.Logger, txReader io.Reader) []map[string]interface{} {
	tx := &auth.StdTx{}
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

func findLog(lf []types.LogFormat, index int) types.LogFormat {
	if len(lf) <= index {
		return types.LogFormat{}
	}
	if l := lf[index]; l.MsgIndex == float64(index) {
		return l
	}
	for _, l := range lf {
		if l.MsgIndex == float64(index) {
			return l
		}
	}
	return types.LogFormat{}
}
