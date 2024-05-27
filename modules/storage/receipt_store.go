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
	"context"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type SaveReceiptsRequest struct {
	Height   uint64
	Receipts []*evmTypes.Receipt
}

type ReceiptStore struct {
	db *mstypes.ReceiptCaches
}

func NewReceiptStore() *ReceiptStore {
	return &ReceiptStore{
		// TODO
		//db: NewReceiptCaches("receiptfiles", 100, 8),
	}
}
func (rs *ReceiptStore) Config(params map[string]interface{}) {
	rs.db = mstypes.NewReceiptCaches(params["storage_receipt_path"].(string), int(params["cache_receipt_size"].(float64)), int(params["cache_exec_concurrency"].(float64)))
}
func (rs *ReceiptStore) Save(ctx context.Context, request *SaveReceiptsRequest, _ *int) error {
	rs.db.Save(request.Height, request.Receipts)
	return nil
}

func (rs *ReceiptStore) Get(ctx context.Context, position *mstypes.Position, receipt **evmTypes.Receipt) error {
	*receipt = rs.db.QueryReceipt(position.Height, position.IdxInBlock)
	return nil
}

func (rs *ReceiptStore) GetBlockReceipts(ctx context.Context, height uint64, receipts *[]*evmTypes.Receipt) error {
	*receipts = rs.db.QueryBlockReceipts(height)
	return nil
}
