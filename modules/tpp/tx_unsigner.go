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
	"context"
	"fmt"
	"math/big"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"

	scommon "github.com/arcology-network/streamer/common"
)

type TxUnsigner struct {
	chainID    *big.Int
	Signer     *evmTypes.Signer
	SignerType uint8
}

// return a Subscriber struct
func NewTxUnsigner() actor.Business {
	unsigner := TxUnsigner{}
	return &unsigner
}

func (c *TxUnsigner) Inputs() ([]string, bool) {
	return []string{scommon.MsgCheckingTxs, scommon.MsgSignerType}, false
}

func (c *TxUnsigner) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgMessager: 1,
		scommon.MsgRpcHash:  1,
	}
}

func (c *TxUnsigner) Config(params map[string]interface{}) {
	c.chainID = params["chain_id"].(*big.Int)
}

func (c *TxUnsigner) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgSignerType, c.ReceivedSIgnType)
	reg.Register(scommon.MsgCheckingTxs, c.ReceivedTxs)
}

func (c *TxUnsigner) ReceivedSIgnType(ctx *actor.ActionContext) error {
	c.SignerType = ctx.Messages[0].Data.(uint8)
	signer := mtypes.MakeSigner(c.SignerType, c.chainID)
	c.Signer = &signer
	return nil
}

func (c *TxUnsigner) ReceivedTxs(ctx *actor.ActionContext) error {

	stdPack := ctx.Messages[0].Data.(*types.StdTransactionPack)
	if c.Signer == nil {
		return nil
	}
	common.ParallelWorker(len(stdPack.Txs), ctx.ExecCtx.Concurrency(), unSignTxs, stdPack.Txs, *c.Signer, c.SignerType)

	ctx.ExecCtx.LogDebug("TxUnsigner.ReceivedTxs", logger.F("count", len(stdPack.Txs)))

	ctx.ExecCtx.Send(scommon.MsgMessager, stdPack)

	if len(stdPack.RequestID) > 0 {
		hash := evmCommon.Hash{}
		if len(stdPack.Txs) > 0 {
			hash = stdPack.Txs[0].TxHash
		}
		ctx.ExecCtx.Send(scommon.MsgRpcHash, hash)
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
			logger.Log.Error(context.Background(), "unSignTxs", "transaction UnSign err", logger.F("err", err))
			continue
		}
		transaction.Signer = signerType
		transactions[i+start] = transaction
	}
}
