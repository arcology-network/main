package txsync

import (
	"fmt"

	"github.com/arcology-network/main/modules/p2p"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type SyncServer struct {
	actor.WorkerThread

	p2pClient P2pClient
}

func NewSyncServer(concurrency int, groupId string) actor.IWorkerEx {
	srv := &SyncServer{
		p2pClient: p2p.NewP2pClient(concurrency, "p2p.client").(P2pClient),
	}
	srv.Set(concurrency, groupId)
	return srv
}

func (srv *SyncServer) Inputs() ([]string, bool) {
	return []string{
		actor.MsgP2pRequest,
	}, false
}

func (srv *SyncServer) Outputs() map[string]int {
	return map[string]int{}
}

func (srv *SyncServer) OnStart() {}

func (srv *SyncServer) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgP2pRequest:
		p2pMessage := msg.Data.(*p2p.P2pMessage)
		msg = p2pMessage.Message
		switch msg.Name {
		case actor.MsgSyncTxRequest:
			height := msg.Data.(uint64)
			var block *mtypes.MonacoBlock
			intf.Router.Call("blockstore", "GetByHeight", &height, &block)
			if block == nil {
				fmt.Printf("[txsync.SyncServer] handle sync tx request from %s, block not found, height = %d\n", p2pMessage.Sender, height)
			} else {
				fmt.Printf("[txsync.SyncServer] handle sync tx request from %s, height = %d, len(txs) = %d\n", p2pMessage.Sender, height, len(block.Txs))
				srv.p2pClient.Response(p2pMessage.Sender, &actor.Message{
					Name: actor.MsgSyncTxResponse,
					Data: block.Txs,
				})
			}
		}
	}
	return nil
}
