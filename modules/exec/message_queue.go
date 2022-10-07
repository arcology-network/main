package exec

import (
	"sync"
)

type MessageQueue struct {
	msgs []interface{}
	lock sync.RWMutex
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		msgs: []interface{}{},
	}
}

func (q *MessageQueue) Insert(msg interface{}) {
	q.lock.Lock()
	defer q.lock.Unlock()

	msgs := make([]interface{}, 0, len(q.msgs)+1)
	msgs = append(msgs, msg)
	msgs = append(msgs, q.msgs...)
	q.msgs = msgs
}

func (q *MessageQueue) Append(msg interface{}) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.msgs = append(q.msgs, msg)
}

func (q *MessageQueue) GetNext() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.msgs) == 0 {
		return nil
	}
	msg := q.msgs[0]
	q.msgs = q.msgs[1:]

	return msg
}
