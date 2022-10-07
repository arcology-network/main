package protocol

import "encoding/binary"

type MsgRouting struct {
	Host string
	Port int
}

func (m MsgRouting) ToMessage() *Message {
	data := make([]byte, 4+len(m.Host))
	binary.LittleEndian.PutUint32(data, uint32(m.Port))
	copy(data[4:], []byte(m.Host))
	return &Message{
		Type: MessageTypeRouting,
		Data: data,
	}
}

func (m *MsgRouting) FromMessage(msg *Message) *MsgRouting {
	m.Port = int(binary.LittleEndian.Uint32(msg.Data[:4]))
	m.Host = string(msg.Data[4:])
	return m
}
