package api

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/gogo/protobuf/proto"

	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
)

func mapSlashingUnjailToSub(msg []byte) (se shared.SubsetEvent, er error) {
	unjail := &slashing.MsgUnjail{}
	if err := proto.Unmarshal(msg, unjail); err != nil {
		return se, errors.New("Not a unjail type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"unjail"},
		Module: "slashing",
		Node:   map[string][]shared.Account{"validator": {{ID: unjail.ValidatorAddr}}},
	}, nil
}
