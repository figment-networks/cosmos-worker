package mapper

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"

	connection "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	"github.com/gogo/protobuf/proto"
)

// IBCConnectionOpenInitToSub transforms ibc.MsgSubmitMisbehaviour sdk messages to SubsetEvent
func IBCConnectionOpenInitToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenInit{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a connection_open_init type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"connection_open_init"},
		Module: "ibc",
	}, nil
}

// IBCConnectionOpenConfirmToSub transforms ibc.MsgSubmitMisbehaviour sdk messages to SubsetEvent
func IBCConnectionOpenConfirmToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenConfirm{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a connection_open_confirm type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"connection_open_confirm"},
		Module: "ibc",
	}, nil
}

// IBCConnectionOpenAckToSub transforms ibc.MsgSubmitMisbehaviour sdk messages to SubsetEvent
func IBCConnectionOpenAckToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenAck{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a connection_open_ack type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"connection_open_ack"},
		Module: "ibc",
	}, nil
}

// IBCConnectionOpenTryToSub transforms ibc.MsgSubmitMisbehaviour sdk messages to SubsetEvent
func IBCConnectionOpenTryToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenTry{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a connection_open_try type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"connection_open_try"},
		Module: "ibc",
	}, nil
}
