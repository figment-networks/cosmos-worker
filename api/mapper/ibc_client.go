package mapper

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"

	client "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	"github.com/gogo/protobuf/proto"
)

// IBCCreateClientToSub transforms ibc.MsgCreateClient sdk messages to SubsetEvent
func IBCCreateClientToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgCreateClient{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a create_client type: " + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"create_client"},
		Module: "ibc",
	}, nil
}

// IBCUpdateClientToSub transforms ibc.MsgUpdateClient sdk messages to SubsetEvent
func IBCUpdateClientToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgUpdateClient{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a update_client type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"update_client"},
		Module: "ibc",
	}, nil
}

// IBCUpgradeClientToSub transforms ibc.MsgUpgradeClient sdk messages to SubsetEvent
func IBCUpgradeClientToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgUpgradeClient{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a upgrade_client type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"upgrade_client"},
		Module: "ibc",
	}, nil
}

// IBCSubmitMisbehaviourToSub transforms ibc.MsgSubmitMisbehaviour sdk messages to SubsetEvent
func IBCSubmitMisbehaviourToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &client.MsgSubmitMisbehaviour{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a submit_misbehaviour type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"submit_misbehaviour"},
		Module: "ibc",
	}, nil
}
