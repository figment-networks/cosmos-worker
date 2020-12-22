package mapper

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
)

// SlashingUnjailToSub transforms slashing.MsgUnjail sdk messages to SubsetEvent
func SlashingUnjailToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	unjail, ok := msg.(slashing.MsgUnjail)
	if !ok {
		return se, errors.New("Not a unjail type")
	}

	return shared.SubsetEvent{
		Type:   []string{"unjail"},
		Module: "slashing",
		Node:   map[string][]shared.Account{"validator": {{ID: unjail.ValidatorAddr}}},
	}, nil
}
