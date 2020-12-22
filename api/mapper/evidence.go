package mapper

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	evidence "github.com/cosmos/cosmos-sdk/x/evidence/types"
)

// EvidenceSubmitEvidenceToSub transforms evidence.MsgSubmitEvidence sdk messages to SubsetEvent
func EvidenceSubmitEvidenceToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	mse, ok := msg.(evidence.MsgSubmitEvidence)
	if !ok {
		return se, errors.New("Not a submit_evidence type")
	}

	return shared.SubsetEvent{
		Type:       []string{"submit_evidence"},
		Module:     "evidence",
		Node:       map[string][]shared.Account{"submitter": {{ID: mse.Submitter}}},
		Additional: map[string][]string{ /*
				"evidence_consensus":       {mse.Evidence.GetConsensusAddress().String()},
				"evidence_height":          {strconv.FormatInt(mse.Evidence.GetHeight(), 10)},
				"evidence_total_power":     {strconv.FormatInt(mse.Evidence.GetTotalPower(), 10)},
				"evidence_validator_power": {strconv.FormatInt(mse.Evidence.GetValidatorPower(), 10)},*/
		},
	}, nil
}
