package protocol

type MsgHandshake struct {
	ID              string
	ConnectionCount byte
}

func (m MsgHandshake) ToMessage() *Message {
	return &Message{
		Type: MessageTypeHandshake,
		Data: append([]byte{m.ConnectionCount}, []byte(m.ID)...),
	}
}

func (m *MsgHandshake) FromMessage(msg *Message) *MsgHandshake {
	m.ID = string(msg.Data[1:])
	m.ConnectionCount = msg.Data[0]
	return m
}
