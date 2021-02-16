package mapper

import (
	"fmt"

	shared "github.com/figment-networks/indexer-manager/structs"

	transfer "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	"github.com/gogo/protobuf/proto"
)

// IBCTransferToSub transforms ibc.MsgTransfer sdk messages to SubsetEvent
func IBCTransferToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &transfer.MsgTransfer{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a transfer type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"transfer"},
		Module: "ibc",
	}, nil
}
