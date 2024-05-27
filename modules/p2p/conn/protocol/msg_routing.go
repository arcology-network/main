/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
