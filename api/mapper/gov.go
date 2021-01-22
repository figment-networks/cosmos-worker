package mapper

import (
	"errors"
	"strconv"

	"github.com/figment-networks/cosmos-worker/api/types"
	shared "github.com/figment-networks/indexer-manager/structs"

	"github.com/cosmos/cosmos-sdk/types"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/gogo/protobuf/proto"
)

// GovDepositToSub transforms gov.MsgDeposit sdk messages to SubsetEvent
func GovDepositToSub(msg []byte, lg types.ABCIMessageLog) (se shared.SubsetEvent, err error) {
	dep := &gov.MsgDeposit{}
	if err := proto.Unmarshal(msg, dep); err != nil {
		return se, errors.New("Not a deposit type" + err.Error())
	}

	se = shared.SubsetEvent{
		Type:       []string{"deposit"},
		Module:     "gov",
		Node:       map[string][]shared.Account{"depositor": {{ID: dep.Depositor}}},
		Additional: map[string][]string{"proposalID": {strconv.FormatUint(dep.ProposalId, 10)}},
	}

	sender := shared.EventTransfer{Account: shared.Account{ID: dep.Depositor}}
	txAmount := map[string]shared.TransactionAmount{}

	for i, coin := range dep.Amount {
		am := shared.TransactionAmount{
			Currency: coin.Denom,
			Numeric:  coin.Amount.BigInt(),
			Text:     coin.Amount.String(),
		}

		sender.Amounts = append(sender.Amounts, am)
		key := "deposit"
		if i > 0 {
			key += "_" + strconv.Itoa(i)
		}

		txAmount[key] = am
	}

	se.Sender = []shared.EventTransfer{sender}
	se.Amount = txAmount

	err = produceTransfers(&se, TransferTypeSend, logf)
	return se, err
}

// GovVoteToSub transforms gov.MsgVote sdk messages to SubsetEvent
func GovVoteToSub(msg []byte) (se shared.SubsetEvent, err error) {
	vote := &gov.MsgVote{}
	if err := proto.Unmarshal(msg, vote); err != nil {
		return se, errors.New("Not a vote type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"vote"},
		Module: "gov",
		Node:   map[string][]shared.Account{"voter": {{ID: vote.Voter}}},
		Additional: map[string][]string{
			"proposalID": {strconv.FormatUint(vote.ProposalId, 10)},
			"option":     {vote.Option.String()},
		},
	}, nil
}

// GovSubmitProposalToSub transforms gov.MsgSubmitProposal sdk messages to SubsetEvent
func GovSubmitProposalToSub(msg []byte, lg types.ABCIMessageLog) (se shared.SubsetEvent, err error) {
	sp := &gov.MsgSubmitProposal{}
	if err := proto.Unmarshal(msg, sp); err != nil {
		return se, errors.New("Not a submit_proposal type" + err.Error())
	}

	se = shared.SubsetEvent{
		Type:   []string{"submit_proposal"},
		Module: "gov",
		Node:   map[string][]shared.Account{"proposer": {{ID: sp.Proposer}}},
	}

	sender := shared.EventTransfer{Account: shared.Account{ID: sp.Proposer}}
	txAmount := map[string]shared.TransactionAmount{}

	for i, coin := range sp.InitialDeposit {
		am := shared.TransactionAmount{
			Currency: coin.Denom,
			Numeric:  coin.Amount.BigInt(),
			Text:     coin.Amount.String(),
		}

		sender.Amounts = append(sender.Amounts, am)
		key := "initial_deposit"
		if i > 0 {
			key += "_" + strconv.Itoa(i)
		}

		txAmount[key] = am
	}
	se.Sender = []shared.EventTransfer{sender}
	se.Amount = txAmount

	se.Additional = map[string][]string{}

	if sp.Content.ProposalRoute() != "" {
		se.Additional["proposal_route"] = []string{sp.Content.ProposalRoute()}
	}
	if sp.Content.ProposalType() != "" {
		se.Additional["proposal_type"] = []string{sp.Content.ProposalType()}
	}
	if sp.Content.GetDescription() != "" {
		se.Additional["descritpion"] = []string{sp.Content.GetDescription()}
	}
	if sp.Content.GetTitle() != "" {
		se.Additional["title"] = []string{sp.Content.GetTitle()}
	}
	if sp.Content.String() != "" {
		se.Additional["content"] = []string{sp.Content.String()}
	}

	err = produceTransfers(&se, TransferTypeSend, logf)
	return se, err
}
