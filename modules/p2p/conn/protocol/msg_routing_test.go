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
