package types

import (
	"time"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	evmTypes "github.com/arcology-network/evm/core/types"
)

type ReceiptCaches struct {
	caches      *DataCache
	db          *RawFile
	concurrency int
}

func NewReceiptCaches(path string, cache int, concurrency int) *ReceiptCaches {
	return &ReceiptCaches{
		caches:      NewDataCache(cache),
		db:          NewRawFiles(path),
		concurrency: concurrency,
	}
}
func (rc *ReceiptCaches) QueryReceipt(height uint64, idx int) *evmTypes.Receipt {
	datas := rc.updateCache(height)
	if datas == nil {
		return nil
	}
	return datas[idx].(*evmTypes.Receipt)
}

func (rc *ReceiptCaches) updateCache(height uint64) []interface{} {
	data, err := rc.db.Read(rc.db.GetFilename(height))
	if err != nil || data == nil {
		return nil
	}
	buffers := [][]byte(codec.Byteset{}.Decode(data).(codec.Byteset))
	datas := make([]interface{}, len(buffers))
	keys := make([]string, len(buffers))
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			receiptobj := evmTypes.Receipt{}
			err = common.GobDecode(buffers[i], &receiptobj)
			if err != nil {
				continue
			}
			datas[i] = &receiptobj
			keys[i] = string(receiptobj.TxHash.Bytes())
		}
	}
	common.ParallelWorker(len(buffers), rc.concurrency, worker)
	rc.caches.Add(height, keys, datas)
	return datas
}

func (rc *ReceiptCaches) Save(height uint64, receipts []*evmTypes.Receipt) ([]string, []time.Duration) {
	tims := make([]time.Duration, 3)
	if len(receipts) == 0 {
		return []string{}, tims
	}
	t0 := time.Now()
	datas := make([]interface{}, len(receipts))
	databyteset := make([][]byte, len(receipts))
	keys := make([]string, len(receipts))
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			receiptRaw, err := common.GobEncode(receipts[i])
			if err != nil {
				continue
			}

			databyteset[i] = receiptRaw
			keys[i] = string(receipts[i].TxHash.Bytes())

			datas[i] = receipts[i]

		}
	}

	common.ParallelWorker(len(receipts), rc.concurrency, worker)
	tims[0] = time.Since(t0)
	t0 = time.Now()
	rc.caches.Add(height, keys, datas)
	tims[1] = time.Since(t0)
	t0 = time.Now()
	rc.db.Write(rc.db.GetFilename(height), codec.Byteset(databyteset).Encode())
	tims[2] = time.Since(t0)
	return keys, tims
}
