package storage

import (
	"github.com/arcology-network/streamer/actor"
)

func init() {
	actor.Factory.Register("gc", NewGc)
	actor.Factory.Register("general_url", func(concurrency int, groupId string) actor.IWorkerEx {
		return NewDBHandler(concurrency, groupId, actor.MsgEuResults, actor.MsgExecuted, actor.MsgGenerationReapingCompleted, actor.MsgBlockEnd, NewGeneralUrl(actor.MsgApcHandle))
	})
}
