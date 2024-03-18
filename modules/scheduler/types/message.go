package types

import (
	eucommon "github.com/arcology-network/eu/common"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type Message struct {
	Message          *eucommon.StandardMessage
	Precedings       *[]*evmCommon.Hash
	PrecedingHash    evmCommon.Hash
	DirectPrecedings *[]*evmCommon.Hash
	// IsSpawned        bool
}

func (msg Message) FromStandardMessage(stdMsg *eucommon.StandardMessage) *Message {
	// return &Message{Message: stdMsg, IsSpawned: false}
	return &Message{Message: stdMsg}
}

type Messages []*Message

func (msgs Messages) FromStandardMessages(stdMsgs []*eucommon.StandardMessage) []*Message {
	schedMegs := make([]*Message, 0, len(stdMsgs))
	for i, schedMeg := range schedMegs {
		schedMegs[i] = schedMeg.FromStandardMessage(stdMsgs[i])
	}
	return schedMegs
}
