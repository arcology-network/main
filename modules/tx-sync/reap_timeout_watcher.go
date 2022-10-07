package txsync

import (
	"fmt"
	"time"

	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/main/modules/p2p"
)

type ReapTimeoutWatcher struct {
	actor.WorkerThread

	batchSize   int
	reapStartCh chan uint64
	reapEndCh   chan uint64
	p2pClient   P2pClient
}

func NewReapTimeoutWatcher(concurrency int, groupId string) actor.IWorkerEx {
	rtw := &ReapTimeoutWatcher{
		batchSize:   1000,
		reapStartCh: make(chan uint64, 1),
		reapEndCh:   make(chan uint64, 1),
		p2pClient:   p2p.NewP2pClient(concurrency, "p2p.client").(P2pClient),
	}
	rtw.Set(concurrency, groupId)
	return rtw
}

func (rtw *ReapTimeoutWatcher) Inputs() ([]string, bool) {
	return []string{
		actor.MsgReapinglist,
		actor.MsgSelectedTx,
		actor.MsgP2pResponse,
	}, false
}

func (rtw *ReapTimeoutWatcher) Outputs() map[string]int {
	return map[string]int{}
}

func (rtw *ReapTimeoutWatcher) Config(params map[string]interface{}) {
	rtw.batchSize = int(params["batch_size"].(float64))
}

func (rtw *ReapTimeoutWatcher) OnStart() {
	go func() {
		for {
			height := <-rtw.reapStartCh
			select {
			case <-rtw.reapEndCh:
				continue
			case <-time.After(time.Second * 3):
				fmt.Printf("[ReapTimeoutWatcher.OnStart] reaping timeout for height %d\n", height)
				rtw.p2pClient.Broadcast(&actor.Message{
					Name: actor.MsgSyncTxRequest,
					Data: height,
				})
				rtw.reapStartCh <- height
			}
		}
	}()
}

func (rtw *ReapTimeoutWatcher) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgReapinglist:
		fmt.Printf("[ReapTimeoutWatcher.OnMessageArrived] reaping started for height %d\n", msg.Height)
		rtw.reapStartCh <- msg.Height
	case actor.MsgSelectedTx:
		fmt.Printf("[ReapTimeoutWatcher.OnMessageArrived] reaping end for height %d\n", msg.Height)
		rtw.reapEndCh <- msg.Height
	case actor.MsgP2pResponse:
		p2pMessage := msg.Data.(*p2p.P2pMessage)
		msg = p2pMessage.Message
		switch msg.Name {
		case actor.MsgSyncTxResponse:
			txs := msg.Data.([][]byte)
			fmt.Printf("[ReapTimeoutWatcher.OnMessageArrived] received %d transactions.\n", len(txs))
			for i := 0; i < len(txs); i += rtw.batchSize {
				var batch [][]byte
				if i+rtw.batchSize < len(txs) {
					batch = txs[i : i+rtw.batchSize]
				} else {
					batch = txs[i:]
				}
				rtw.MsgBroker.Send(actor.MsgTxBlocks, batch)
			}
		}
	}
	return nil
}
