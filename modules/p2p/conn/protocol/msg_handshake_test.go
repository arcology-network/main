package protocol

import "testing"

func TestMsgHandshakeToMessageFromMessage(t *testing.T) {
	m := MsgHandshake{
		ID:              "test",
		ConnectionCount: 4,
	}

	msg := m.ToMessage()
	var m2 MsgHandshake
	m2.FromMessage(msg)

	if m.ID != m2.ID ||
		m.ConnectionCount != m2.ConnectionCount {
		t.Error("Fail")
	}
}
