package core

import (
	"fmt"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/mhasher"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/log"
	"go.uber.org/zap"
)

type CalculateTxHash struct {
	actor.WorkerThread
}

//return a Subscriber struct
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
	roothash := ethCommon.Hash{}
	if len(txs) > 0 {
		begintime1 := time.Now()
		roothash = mhasher.GetTxsHash(txs)
		c.AddLog(log.LogLevel_Debug, "calculate txroothash", zap.Duration("times", time.Since(begintime1)))
	}
	c.AddLog(log.LogLevel_CheckPoint, "send txroothash", zap.Int("txsnum", len(txs)), zap.String("roothash", fmt.Sprintf("%x", roothash)))
	c.MsgBroker.Send(actor.MsgTxHash, &roothash)
	return nil
}
