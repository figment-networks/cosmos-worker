package mapper

import (
	"errors"

	shared "github.com/figment-networks/indexer-manager/structs"

	channel "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/gogo/protobuf/proto"
)

// IBCChannelOpenInitToSub transforms ibc.MsgChannelOpenInit sdk messages to SubsetEvent
func IBCChannelOpenInitToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgChannelOpenInit{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, errors.New("Not a channel_open_init type" + err.Error())
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_open_init"},
		Module: "ibc",
	}, nil
}
