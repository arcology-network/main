package types

import (
	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	"github.com/arcology-network/common-lib/types"
)

type Message struct {
	Message          *types.StandardMessage
	Precedings       *[]*ethCommon.Hash
	PrecedingHash    ethCommon.Hash
	DirectPrecedings *[]*ethCommon.Hash
	IsSpawned        bool
}

func (msg Message) FromStandardMessage(stdMsg *types.StandardMessage) *Message {
	return &Message{Message: stdMsg, IsSpawned: false}
}

type Messages []*Message

func (msgs Messages) FromStandardMessages(stdMsgs []*types.StandardMessage) []*Message {
	schedMegs := make([]*Message, 0, len(stdMsgs))
	for i, schedMeg := range schedMegs {
		schedMegs[i] = schedMeg.FromStandardMessage(stdMsgs[i])
	}
	return schedMegs
}
