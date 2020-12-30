package api

import (
	"errors"
	"strconv"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func mapGovDepositToSub(msg []byte) (se shared.SubsetEvent, er error) {
	dep := &gov.MsgDeposit{}
	if err := proto.Unmarshal(msg, dep); err != nil {
		return se, errors.New("Not a deposit type" + err.Error())
	}

	evt := shared.SubsetEvent{
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

	evt.Sender = []shared.EventTransfer{sender}
	evt.Amount = txAmount

	return evt, nil
}

func mapGovVoteToSub(msg []byte) (se shared.SubsetEvent, er error) {
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

func mapGovSubmitProposalToSub(msg []byte) (se shared.SubsetEvent, er error) {
	sp := &gov.MsgSubmitProposal{}
	if err := proto.Unmarshal(msg, sp); err != nil {
		return se, errors.New("Not a submit_proposal type" + err.Error())
	}

	evt := shared.SubsetEvent{
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
	evt.Sender = []shared.EventTransfer{sender}
	evt.Amount = txAmount

	// TODO(lukanus): Any description of the contents of that is not available. Cosmos team is not responsive
	/*
		//	evt.Additional = map[string][]string{}
			if sp.Content.ProposalRoute() != "" {
				evt.Additional["proposal_route"] = []string{sp.Content.ProposalRoute()}
			}
			if sp.Content.ProposalType() != "" {
				evt.Additional["proposal_type"] = []string{sp.Content.ProposalType()}
			}
			if sp.Content.GetDescription() != "" {
				evt.Additional["descritpion"] = []string{sp.Content.GetDescription()}
			}
			if sp.Content.GetTitle() != "" {
				evt.Additional["title"] = []string{sp.Content.GetTitle()}
			}
			if sp.Content.String() != "" {
				evt.Additional["content"] = []string{sp.Content.String()}
			}
	*/
	return evt, nil
}
