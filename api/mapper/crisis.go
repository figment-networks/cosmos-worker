package mapper

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	crisis "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

// CrisisVerifyInvariantToSub transforms crisis.MsgVerifyInvariant sdk messages to SubsetEvent
func CrisisVerifyInvariantToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	mvi, ok := msg.(crisis.MsgVerifyInvariant)
	if !ok {
		return se, errors.New("Not a verify_invariant type")
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
