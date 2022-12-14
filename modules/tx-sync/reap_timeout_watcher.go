package txsync

import (
	"fmt"
	"time"

	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/main/modules/p2p"
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
	return map[string]int{
		actor.MsgTxBlocks: 1,
	}
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
			case <-time.After(time.Second * 30):
				rtw.AddLog(log.LogLevel_Info, fmt.Sprintf("[ReapTimeoutWatcher.OnStart] reaping timeout for height %d\n", height))
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
		rtw.AddLog(log.LogLevel_Info, fmt.Sprintf("[ReapTimeoutWatcher.OnMessageArrived] reaping started for height %d\n", msg.Height))
		rtw.reapStartCh <- msg.Height
	case actor.MsgSelectedTx:
		rtw.AddLog(log.LogLevel_Info, fmt.Sprintf("[ReapTimeoutWatcher.OnMessageArrived] reaping end for height %d\n", msg.Height))
		rtw.reapEndCh <- msg.Height
	case actor.MsgP2pResponse:
		p2pMessage := msg.Data.(*p2p.P2pMessage)
		msg = p2pMessage.Message
		switch msg.Name {
		case actor.MsgSyncTxResponse:
			txs := msg.Data.([][]byte)
			rtw.AddLog(log.LogLevel_Info, fmt.Sprintf("[ReapTimeoutWatcher.OnMessageArrived] received %d transactions.\n", len(txs)))
			for i := 0; i < len(txs); i += rtw.batchSize {
				var batch [][]byte
				if i+rtw.batchSize < len(txs) {
					batch = txs[i : i+rtw.batchSize]
				} else {
					batch = txs[i:]
				}
				rtw.MsgBroker.Send(actor.MsgTxBlocks, &cmntyp.IncomingTxs{
					Txs: batch,
					Src: cmntyp.NewTxSource(cmntyp.TxSourceMonacoP2p, p2pMessage.Sender),
				})
			}
		}
	}
	return nil
}
