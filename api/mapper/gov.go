package mapper

import (
	"errors"
	"strconv"

	"github.com/figment-networks/cosmos-worker/api/types"
	shared "github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gov "github.com/cosmos/cosmos-sdk/x/gov"
)

// GovDepositToSub transforms gov.MsgDeposit sdk messages to SubsetEvent
func GovDepositToSub(msg sdk.Msg, logf types.LogFormat) (se shared.SubsetEvent, err error) {
	dep, ok := msg.(gov.MsgDeposit)
	if !ok {
		return se, errors.New("Not a deposit type")
	}

	se = shared.SubsetEvent{
		Type:       []string{"deposit"},
		Module:     "gov",
		Node:       map[string][]shared.Account{"depositor": {{ID: dep.Depositor.String()}}},
		Additional: map[string][]string{"proposalID": {strconv.FormatUint(dep.ProposalID, 10)}},
	}

	sender := shared.EventTransfer{Account: shared.Account{ID: dep.Depositor.String()}}
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
func GovVoteToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	vote, ok := msg.(gov.MsgVote)
	if !ok {
		return se, errors.New("Not a vote type")
	}

	return shared.SubsetEvent{
		Type:   []string{"vote"},
		Module: "gov",
		Node:   map[string][]shared.Account{"voter": {{ID: vote.Voter.String()}}},
		Additional: map[string][]string{
			"proposalID": {strconv.FormatUint(vote.ProposalID, 10)},
			"option":     {vote.Option.String()},
		},
	}, nil
}

// GovSubmitProposalToSub transforms gov.MsgSubmitProposal sdk messages to SubsetEvent
func GovSubmitProposalToSub(msg sdk.Msg, logf types.LogFormat) (se shared.SubsetEvent, err error) {
	sp, ok := msg.(gov.MsgSubmitProposal)
	if !ok {
		return se, errors.New("Not a submit_proposal type")
	}

	se = shared.SubsetEvent{
		Type:   []string{"submit_proposal"},
		Module: "gov",
		Node:   map[string][]shared.Account{"proposer": {{ID: sp.Proposer.String()}}},
	}

	sender := shared.EventTransfer{Account: shared.Account{ID: sp.Proposer.String()}}
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
