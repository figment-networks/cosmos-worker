package mapper

import (
	"errors"
	"strconv"

	shared "github.com/figment-networks/indexer-manager/structs"

	"github.com/cosmos/cosmos-sdk/x/evidence/exported"
	evidence "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/gogo/protobuf/proto"
)

// EvidenceSubmitEvidenceToSub transforms evidence.MsgSubmitEvidence sdk messages to SubsetEvent
func EvidenceSubmitEvidenceToSub(msg []byte) (se shared.SubsetEvent, er error) {
	mse := &evidence.MsgSubmitEvidence{}
	if err := proto.Unmarshal(msg, mse); err != nil {
		return se, errors.New("Not a submit_evidence type" + err.Error())
	}

	se = shared.SubsetEvent{
		Type:   []string{"submit_evidence"},
		Module: "evidence",
		Node:   map[string][]shared.Account{"submitter": {{ID: mse.Submitter}}},
		Additional: map[string][]string{
			"evidence_height": {strconv.FormatInt(mse.GetEvidence().GetHeight(), 10)},
		},
	}

	validatorEvi, ok := mse.Evidence.GetCachedValue().(exported.ValidatorEvidence)
	if !ok {
		return se, nil
	}

	se.Additional["evidence_consensus"] = []string{validatorEvi.GetConsensusAddress().String()}
	se.Additional["evidence_total_power"] = []string{strconv.FormatInt(validatorEvi.GetTotalPower(), 10)}
	se.Additional["evidence_validator_power"] = []string{strconv.FormatInt(validatorEvi.GetValidatorPower(), 10)}
	return se, nil
}
