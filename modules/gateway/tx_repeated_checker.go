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

package gateway

import (
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/components/storage"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
)

type TxRepeatedChecker struct {
	checklist *storage.CheckedList
	waits     int64
	maxSize   int
}

// return a Subscriber struct
func NewTxRepeatedChecker() actor.Business {
	receiver := TxRepeatedChecker{}
	return &receiver
}

func (r *TxRepeatedChecker) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgTxLocalsUnChecked,
		scommon.MsgTxBlocks,
	}, false
}

func (r *TxRepeatedChecker) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgCheckedTxs: 100,
		scommon.MsgTxLocals:   100,
	}
}

func (r *TxRepeatedChecker) Config(params map[string]interface{}) {
	r.waits = int64(params["wait_seconds"].(int))
	r.maxSize = params["max_txs_num"].(int)
	r.checklist = storage.NewCheckList(r.waits)
}

func (r *TxRepeatedChecker) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgTxLocalsUnChecked, r.ReceivedTxLocalsUnChecked)
	reg.Register(scommon.MsgTxBlocks, r.ReceivedTxBlocks)
}

func (r *TxRepeatedChecker) ReceivedTxLocalsUnChecked(ctx *actor.ActionContext) error {
	r.checkRepeated(ctx, ctx.Messages[0].Data.(*types.IncomingTxs), types.TxFrom_Local)
	return nil
}

func (r *TxRepeatedChecker) ReceivedTxBlocks(ctx *actor.ActionContext) error {
	r.checkRepeated(ctx, ctx.Messages[0].Data.(*types.IncomingTxs), types.TxFrom_Block)
	return nil
}

func (r *TxRepeatedChecker) checkRepeated(ctx *actor.ActionContext, txspack *types.IncomingTxs, from byte) {
	txs := txspack.Txs
	txLen := len(txs)
	checkedTxs := make([][]byte, 0, txLen)
	ctx.ExecCtx.LogDebug("checkRepeated")

	bypassRepeatCheck := txspack.Src.BypassRepeatCheck()
	for i := range txs {
		isExist := r.checklist.ExistTx(txs[i], from)
		if !isExist || bypassRepeatCheck {
			tx := txs[i]
			sendingTx := make([]byte, len(tx)+1)
			bz := 0
			bz += copy(sendingTx[bz:], []byte{from})
			bz += copy(sendingTx[bz:], tx)

			checkedTxs = append(checkedTxs, sendingTx)
		}
	}

	//to other node with consensus
	if from == types.TxFrom_Local {
		ctx.ExecCtx.Send(scommon.MsgTxLocals, txs)
	}

	//to tpp with stream,split pack
	sendingTxs := make([][]byte, 0, r.maxSize)
	for i := range checkedTxs {
		if len(sendingTxs) >= r.maxSize {
			ctx.ExecCtx.Send(scommon.MsgCheckedTxs, &types.IncomingTxs{
				Txs:       sendingTxs,
				Src:       txspack.Src,
				RequestID: txspack.RequestID,
			})
			sendingTxs = make([][]byte, 0, r.maxSize)
		} else {
			sendingTxs = append(sendingTxs, checkedTxs[i])
		}
	}
	ctx.ExecCtx.LogDebug("TxRepeatedChecker.checkRepeated", logger.F("sendingTxs", len(sendingTxs)))
	if len(sendingTxs) > 0 {
		ctx.ExecCtx.Send(scommon.MsgCheckedTxs, &types.IncomingTxs{
			Txs:       sendingTxs,
			Src:       txspack.Src,
			RequestID: txspack.RequestID,
		})
	}

}
