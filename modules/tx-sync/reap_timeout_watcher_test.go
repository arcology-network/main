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
