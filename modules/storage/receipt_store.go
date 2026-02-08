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

package storage

import (
	mstypes "github.com/arcology-network/main/modules/storage/types"
	"github.com/arcology-network/streamer/actor"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type SaveReceiptsRequest struct {
	Height   uint64
	Receipts []*evmTypes.Receipt
}

type ReceiptStore struct {
	db *mstypes.ReceiptCaches
}

func NewReceiptStore() actor.Business {
	return &ReceiptStore{}
}

func (rs *ReceiptStore) Config(params map[string]interface{}) {
	rs.db = mstypes.NewReceiptCaches(params["storage_receipt_path"].(string), params["cache_receipt_size"].(int), params["cache_exec_concurrency"].(int))
}

func (rs *ReceiptStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (rs *ReceiptStore) Outputs() map[string]int {
	return map[string]int{}
}

func (rs *ReceiptStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Save", rs.Save)
	reg.Register("Get", rs.Get)
	reg.Register("GetBlockReceipts", rs.GetBlockReceipts)
}

func (rs *ReceiptStore) RpcConfig() (string, int) {
	return "receiptstore", 20
}

func (rs *ReceiptStore) Save(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*SaveReceiptsRequest)
	rs.db.Save(request.Height, request.Receipts)
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (rs *ReceiptStore) Get(ctx *actor.ActionContext) error {
	position := ctx.RPC.Request.(*mstypes.Position)
	ctx.ExecCtx.SendRpcResponse("", rs.db.QueryReceipt(position.Height, position.IdxInBlock))
	return nil
}

func (rs *ReceiptStore) GetBlockReceipts(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(uint64)
	ctx.ExecCtx.SendRpcResponse("", rs.db.QueryBlockReceipts(height))
	return nil
}
