package core

import (
	"fmt"
	"time"

	"github.com/arcology-network/common-lib/mhasher"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type CalculateTxHash struct {
	actor.WorkerThread
}

// return a Subscriber struct
func NewCalculateTxHash(concurrency int, groupid string) actor.IWorkerEx {
	in := CalculateTxHash{}
	in.Set(concurrency, groupid)
	return &in
}

func (c *CalculateTxHash) Inputs() ([]string, bool) {
	return []string{actor.MsgSelectedTx}, false
}

func (c *CalculateTxHash) Outputs() map[string]int {
	return map[string]int{
		actor.MsgTxHash: 1,
	}
}

func (c *CalculateTxHash) OnStart() {
}

func (c *CalculateTxHash) OnMessageArrived(msgs []*actor.Message) error {
	datas := msgs[0].Data.(types.Txs)
	txs := datas.Data
	c.CheckPoint("received selectedtx")
	roothash := evmCommon.Hash{}
	if len(txs) > 0 {
		begintime1 := time.Now()
		roothash = mhasher.GetTxsHash(txs)
		c.AddLog(log.LogLevel_Debug, "calculate txroothash", zap.Duration("times", time.Since(begintime1)))
	}
	c.CheckPoint("send txroothash", zap.Int("txsnum", len(txs)), zap.String("roothash", fmt.Sprintf("%x", roothash)))
	c.MsgBroker.Send(actor.MsgTxHash, &roothash)
	return nil
}
