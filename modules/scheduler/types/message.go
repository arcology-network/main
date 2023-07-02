package types

import (
	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/arcology-network/evm/common"
)

type Message struct {
	Message          *types.StandardMessage
	Precedings       *[]*evmCommon.Hash
	PrecedingHash    evmCommon.Hash
	DirectPrecedings *[]*evmCommon.Hash
	// IsSpawned        bool
}

func (msg Message) FromStandardMessage(stdMsg *types.StandardMessage) *Message {
	// return &Message{Message: stdMsg, IsSpawned: false}
	return &Message{Message: stdMsg}
}

type Messages []*Message

func (msgs Messages) FromStandardMessages(stdMsgs []*types.StandardMessage) []*Message {
	schedMegs := make([]*Message, 0, len(stdMsgs))
	for i, schedMeg := range schedMegs {
		schedMegs[i] = schedMeg.FromStandardMessage(stdMsgs[i])
	}
	return schedMegs
}
