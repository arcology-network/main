package storage

import (
	"context"

	ethtyp "github.com/HPISTechnologies/3rd-party/eth/types"
	mstypes "github.com/HPISTechnologies/main/modules/storage/types"
)

type SaveReceiptsRequest struct {
	Height   uint64
	Receipts []*ethtyp.Receipt
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

func (rs *ReceiptStore) Get(ctx context.Context, position *mstypes.Position, receipt **ethtyp.Receipt) error {
	*receipt = rs.db.QueryReceipt(position.Height, position.IdxInBlock)
	return nil
}
