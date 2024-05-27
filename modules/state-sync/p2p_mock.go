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

package statesync

import (
	"github.com/arcology-network/main/modules/p2p"
	"github.com/arcology-network/streamer/actor"
)

type switchMock struct {
	clients map[string]*p2pClientMock
}

func newSwitchMock() *switchMock {
	return &switchMock{
		clients: make(map[string]*p2pClientMock),
	}
}

func (m *switchMock) add(client *p2pClientMock) {
	m.clients[client.id] = client
	client.sw = m
}

func (m *switchMock) broadcast(sender string, msg *actor.Message) {
	for id, client := range m.clients {
		if id != sender {
			client.consumer.OnMessageArrived([]*actor.Message{msg})
		}
	}
}

func (m *switchMock) send(sender, receiver string, msg *actor.Message) {
	m.clients[receiver].consumer.OnMessageArrived([]*actor.Message{msg})
}

type p2pClientMock struct {
	id       string
	sw       *switchMock
	consumer actor.IWorker
}

func newP2pClientMock(id string, consumer actor.IWorker) *p2pClientMock {
	return &p2pClientMock{
		id:       id,
		consumer: consumer,
	}
}

func (m *p2pClientMock) ID() string {
	return m.id
}

func (m *p2pClientMock) Broadcast(msg *actor.Message) {
	m.sw.broadcast(m.id, &actor.Message{
		Name: actor.MsgP2pRequest,
		Data: &p2p.P2pMessage{
			Sender:  m.id,
			Message: msg,
		},
	})
}

func (m *p2pClientMock) Request(peer string, msg *actor.Message) {
	m.sw.send(m.id, peer, &actor.Message{
		Name: actor.MsgP2pRequest,
		Data: &p2p.P2pMessage{
			Sender:  m.id,
			Message: msg,
		},
	})
}

func (m *p2pClientMock) Response(peer string, msg *actor.Message) {
	m.sw.send(m.id, peer, &actor.Message{
		Name: actor.MsgP2pResponse,
		Data: &p2p.P2pMessage{
			Sender:  m.id,
			Message: msg,
		},
	})
}

func (m *p2pClientMock) OnConnClosed(cb func(id string)) {}
