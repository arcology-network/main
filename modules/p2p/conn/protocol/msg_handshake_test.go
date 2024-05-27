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
