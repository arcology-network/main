package types

import (
	"github.com/arcology-network/common-lib/types"
)

type Queue struct {
	txs []*types.StandardMessage
	ids []uint32
	idx int
}

func NewQueue() *Queue {
	return &Queue{
		txs: []*types.StandardMessage{},
		ids: []uint32{},
		idx: 0,
	}
}

func (q *Queue) Reset(msgs []*types.StandardMessage, txids []uint32) {
	q.idx = 0
	txs := make([]*types.StandardMessage, 0, len(msgs))
	txs = append(txs, msgs...)
	q.txs = txs

	ids := make([]uint32, 0, len(txids))
	ids = append(ids, txids...)
	q.ids = ids
}

func (q *Queue) Insert(msg *types.StandardMessage, id uint32) {
	if q.idx == 0 {
		txs := make([]*types.StandardMessage, 0, len(q.txs)+1)
		txs = append(txs, msg)
		txs = append(txs, q.txs...)
		q.txs = txs

		ids := make([]uint32, 0, len(q.ids)+1)
		ids = append(ids, id)
		ids = append(ids, q.ids...)
		q.ids = ids
	} else {
		q.txs[q.idx-1] = msg
		q.ids[q.idx-1] = id
		q.idx = q.idx - 1
	}
}

func (q *Queue) GetNext() (*types.StandardMessage, uint32) {
	if q.idx >= len(q.txs) {
		return nil, 0
	}
	msg := q.txs[q.idx]
	id := q.ids[q.idx]
	q.idx = q.idx + 1
	return msg, id
}
