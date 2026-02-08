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
	"github.com/arcology-network/common-lib/types"
	tppTypes "github.com/arcology-network/main/modules/tpp/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
)

type TxReceiver struct {
}

// return a Subscriber struct
func NewTxReceiver() actor.Business {
	receiver := TxReceiver{}
	return &receiver
}

func (r *TxReceiver) Inputs() ([]string, bool) {
	return []string{scommon.MsgCheckedTxs}, false
}

func (r *TxReceiver) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgCheckingTxs: 10,
	}
}

func (r *TxReceiver) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgCheckedTxs, r.processTxs)
}

func (r *TxReceiver) processTxs(ctx *actor.ActionContext) error {
	txs := ctx.Messages[0].Data.(*types.IncomingTxs)

	ctx.ExecCtx.LogDebug("start processTxs Txs", logger.F("count", len(txs.Txs)))

	pack := tppTypes.NewPack(txs, ctx.ExecCtx.Concurrency())

	ctx.ExecCtx.Send(scommon.MsgCheckingTxs, pack)

	return nil
}
