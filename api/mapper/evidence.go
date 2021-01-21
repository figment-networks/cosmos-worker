package mapper

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"

	evidence "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/gogo/protobuf/proto"
)

// EvidenceSubmitEvidenceToSub transforms evidence.MsgSubmitEvidence sdk messages to SubsetEvent
func EvidenceSubmitEvidenceToSub(msg []byte) (se shared.SubsetEvent, er error) {
	mse := &evidence.MsgSubmitEvidence{}
	if err := proto.Unmarshal(msg, mse); err != nil {
		return se, errors.New("Not a submit_evidence type" + err.Error())
	}

	// TODO(lukanus): Any description of the contents of that is not available. Cosmos team is not responsive
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
