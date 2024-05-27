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

package txsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/arcology-network/main/modules/p2p"
	"github.com/arcology-network/streamer/actor"
	brokerpk "github.com/arcology-network/streamer/broker"
	intf "github.com/arcology-network/streamer/interface"
)

type p2pMock struct{}

func (mock *p2pMock) Send(_ context.Context, request *p2p.P2pSendRequest, _ *int) {
	fmt.Printf("[p2pMock.Send] request = %v\n", request)
}

func rpcSetup() {
	intf.RPCCreator = func(string, string, []string, []interface{}, []interface{}) {}
	intf.Router.Register("p2p", &p2pMock{}, "", "")
}

func TestReapTimeoutWatcher(t *testing.T) {
	rpcSetup()
	broker := brokerpk.NewStatefulStreamer()
	rtw := NewReapTimeoutWatcher(1, "tester").(*ReapTimeoutWatcher)
	rtw.Init("watcher", broker)
	client := p2p.NewP2pClient(1, "client")
	client.Init("client", broker)
	broker.RegisterProducer(brokerpk.NewDefaultProducer("watcher", []string{actor.MsgP2pSent}, []int{1}))
	broker.Serve()
	rtw.OnStart()

	rtw.OnMessageArrived([]*actor.Message{
		{
			Name:   actor.MsgReapinglist,
			Height: 1,
		},
	})
	time.Sleep(8 * time.Second)
	rtw.OnMessageArrived([]*actor.Message{
		{
			Name:   actor.MsgSelectedTx,
			Height: 1,
		},
	})
	time.Sleep(3 * time.Second)
}
