package api

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	crisis "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

func mapCrisisVerifyInvariantToSub(msg []byte) (se shared.SubsetEvent, er error) {
	mvi := &crisis.MsgVerifyInvariant{}
	if err := proto.Unmarshal(msg, mvi); err != nil {
		return se, errors.New("Not a crisis type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"verify_invariant"},
		Module: "crisis",
		Sender: []shared.EventTransfer{{
			Account: shared.Account{ID: mvi.Sender},
		}},
		Additional: map[string][]string{
			"invariant_route":       {mvi.InvariantRoute},
			"invariant_module_name": {mvi.InvariantModuleName},
		},
	}, nil
}
