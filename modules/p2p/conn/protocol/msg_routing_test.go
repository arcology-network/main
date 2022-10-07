package protocol

import "testing"

func TestMsgRoutingToMessageFromMessage(t *testing.T) {
	m := MsgRouting{
		Host: "127.0.0.1",
		Port: 9191,
	}

	msg := m.ToMessage()
	var m2 MsgRouting
	m2.FromMessage(msg)

	if m.Host != m2.Host || m.Port != m2.Port {
		t.Error("Fail")
	}
}
