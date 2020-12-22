package api

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	evidence "github.com/cosmos/cosmos-sdk/x/evidence/types"
)

func mapEvidenceSubmitEvidenceToSub(msg []byte) (se shared.SubsetEvent, er error) {
	mse := &evidence.MsgSubmitEvidence{}
	if err := proto.Unmarshal(msg, mse); err != nil {
		return se, errors.New("Not a submit_evidence type" + err.Error())
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
