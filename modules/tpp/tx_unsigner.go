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
