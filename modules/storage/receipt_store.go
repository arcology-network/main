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
