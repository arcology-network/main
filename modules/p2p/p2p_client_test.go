package p2p

import (
	"testing"

	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
)

func TestMessageToPackages(t *testing.T) {
	data, _ := (&actor.Message{
		Name: actor.MsgP2pRequest,
		Data: &P2pMessage{
			Sender: "node0",
			Message: &actor.Message{
				Name: actor.MsgSyncStatusRequest,
			},
		},
	}).Encode()
	t.Log(data)

	packages := protocol.Message{
		Type: protocol.MessageTypeClientBroadcast,
		Data: data,
	}.ToPackages()
	t.Log(packages)
}
