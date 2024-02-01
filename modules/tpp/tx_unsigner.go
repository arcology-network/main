package tpp

import (
	"fmt"
	"math/big"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type TxUnsigner struct {
	actor.WorkerThread
	chainID    *big.Int
	Signer     *evmTypes.Signer
	SignerType uint8
}

// return a Subscriber struct
func NewTxUnsigner(concurrency int, groupid string) actor.IWorkerEx {
	unsigner := TxUnsigner{}
	unsigner.Set(concurrency, groupid)
	return &unsigner
}

func (c *TxUnsigner) Inputs() ([]string, bool) {
	return []string{actor.MsgCheckingTxs, actor.MsgSignerType}, false
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
		case actor.MsgSignerType:
			c.SignerType = v.Data.(uint8)
			signer := mtypes.MakeSigner(c.SignerType, c.chainID)
			c.Signer = &signer
		case actor.MsgCheckingTxs:
			stdPack := v.Data.(*types.StdTransactionPack)
			if c.Signer == nil {
				return nil
			}
			common.ParallelWorker(len(stdPack.Txs), c.Concurrency, unSignTxs, stdPack.Txs, *c.Signer, c.SignerType)

			c.MsgBroker.Send(actor.MsgMessager, stdPack)

			if stdPack.TxHashChan != nil {
				if len(stdPack.Txs) > 0 {
					stdPack.TxHashChan <- stdPack.Txs[0].TxHash
				} else {
					stdPack.TxHashChan <- evmCommon.Hash{}
				}
			}
		}
	}
	return nil
}

func unSignTxs(start, end, idx int, args ...interface{}) {
	transactions := args[0].([]interface{})[0].(types.StandardTransactions)
	signer := args[0].([]interface{})[1].(evmTypes.Signer)
	signerType := args[0].([]interface{})[2].(uint8)
	// logg := args[0].([]interface{})[1].(*actor.WorkerThreadLogger)

	for i, transaction := range transactions[start:end] {
		if transaction.NativeTransaction == nil {
			continue
		}
		if err := transaction.UnSign(signer); err != nil {
			fmt.Printf("========================UnSign err:%v\n", err)
			continue
		}
		transaction.Signer = signerType
		transactions[i+start] = transaction
	}
}

// func (c *TxUnsigner) unSignTxs(ctxs []*tpptypes.CheckingTx) []*types.StandardMessage {
// 	txLen := len(ctxs)
// 	threads := c.Concurrency
// 	var step = int(math.Max(float64(txLen/threads), float64(txLen%threads)))
// 	wg := sync.WaitGroup{}
// 	c.AddLog(log.LogLevel_Debug, "start decodeing txs>>>>>>>>>>>>>>>>>>", zap.Int("txLen", txLen))
// 	messages := make([]*types.StandardMessage, len(ctxs))
// 	for counter := 0; counter <= threads; counter++ {
// 		begin := counter * step
// 		end := int(math.Min(float64(begin+step), float64(txLen)))
// 		wg.Add(1)
// 		go func(begin int, end int, id int) {
// 			for i := begin; i < end; i++ {
// 				err := ctxs[i].UnSign(c.chainID)
// 				if err != nil {
// 					fmt.Printf("========================UnSign err:%v\n", err)
// 					messages[i] = nil
// 					continue
// 				}
// 				messages[i] = &ctxs[i].Message
// 			}
// 			wg.Done()
// 		}(begin, end, counter)
// 		if txLen == end {
// 			break
// 		}
// 	}
// 	wg.Wait()
// 	c.AddLog(log.LogLevel_Debug, "decodeing txs completed <<<<<<<<<<<<<<<<<<<<<<<<<<<", zap.Int("txLen", txLen))
// 	return messages
// }
