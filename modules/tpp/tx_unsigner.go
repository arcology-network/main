package tpp

import (
	"fmt"
	"math"
	"math/big"
	"sync"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	tpptypes "github.com/arcology-network/main/modules/tpp/types"
	"go.uber.org/zap"
)

type TxUnsigner struct {
	actor.WorkerThread
	chainID *big.Int
}

// return a Subscriber struct
func NewTxUnsigner(concurrency int, groupid string) actor.IWorkerEx {
	unsigner := TxUnsigner{}
	unsigner.Set(concurrency, groupid)
	return &unsigner
}

func (c *TxUnsigner) Inputs() ([]string, bool) {
	return []string{actor.MsgCheckingTxs}, false
}

func (c *TxUnsigner) Outputs() map[string]int {
	return map[string]int{
		actor.MsgMessager: 1,
	}
}

func (c *TxUnsigner) Config(params map[string]interface{}) {
	c.chainID = params["chain_id"].(*big.Int)
}

func (c *TxUnsigner) OnStart() {}

func (c *TxUnsigner) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgCheckingTxs:
			checkingTxsPack := v.Data.(*tpptypes.CheckingTxsPack)
			filteredTxs := make([]*tpptypes.CheckingTx, 0, len(checkingTxsPack.Txs))

			for i := range checkingTxsPack.Txs {

				if checkingTxsPack.Txs[i] == nil {
					c.AddLog(log.LogLevel_Error, "checkingTxs is nil >>>>>>>>>>>>>>>>>>", zap.Int("idx", i))
					continue
				}
				filteredTxs = append(filteredTxs, checkingTxsPack.Txs[i])
			}

			messages := c.unSignTxs(filteredTxs)
			txHashes := make([]ethCommon.Hash, len(messages))
			for i := range messages {
				txHashes[i] = messages[i].TxHash
			}
			c.MsgBroker.Send(actor.MsgMessager, &types.IncomingMsgs{
				Msgs: messages,
				Src:  checkingTxsPack.Src,
			})
			if checkingTxsPack.TxHashChan != nil {
				checkingTxsPack.TxHashChan <- txHashes[0]
			}
		}
	}
	return nil
}

func (c *TxUnsigner) unSignTxs(ctxs []*tpptypes.CheckingTx) []*types.StandardMessage {
	txLen := len(ctxs)
	threads := c.Concurrency
	var step = int(math.Max(float64(txLen/threads), float64(txLen%threads)))
	wg := sync.WaitGroup{}
	c.AddLog(log.LogLevel_Debug, "start decodeing txs>>>>>>>>>>>>>>>>>>", zap.Int("txLen", txLen))
	messages := make([]*types.StandardMessage, len(ctxs))
	for counter := 0; counter <= threads; counter++ {
		begin := counter * step
		end := int(math.Min(float64(begin+step), float64(txLen)))
		wg.Add(1)
		go func(begin int, end int, id int) {
			for i := begin; i < end; i++ {
				err := ctxs[i].UnSign(c.chainID)
				if err != nil {
					fmt.Printf("========================UnSign err:%v\n", err)
				}
				messages[i] = &ctxs[i].Message
			}
			wg.Done()
		}(begin, end, counter)
		if txLen == end {
			break
		}
	}
	wg.Wait()
	c.AddLog(log.LogLevel_Debug, "decodeing txs completed <<<<<<<<<<<<<<<<<<<<<<<<<<<", zap.Int("txLen", txLen))
	return messages
}
