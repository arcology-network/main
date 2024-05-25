package storage

import (
	"github.com/arcology-network/streamer/actor"
)

func init() {
	actor.Factory.Register("gc", NewGc)
	actor.Factory.Register("general_url", func(concurrency int, groupId string) actor.IWorkerEx {
		return NewDBHandler(concurrency, groupId, actor.MsgEuResults, actor.MsgExecuted, actor.MsgGenerationReapingCompleted, actor.MsgBlockEnd, actor.MsgInitDBGeneral,
			NewGeneralUrl(actor.MsgApcHandle, actor.MsgGeneralDB, actor.MsgGeneralCompleted, actor.MsgGeneralPrecommit, actor.MsgGeneralCommit))
	})
	actor.Factory.Register("general_url_async", func(concurrency int, groupId string) actor.IWorkerEx {
		return NewDBHandlerAsync(concurrency, groupId, actor.MsgGeneralDB, actor.MsgGeneralPrecommit, actor.MsgGeneralCommit, actor.MsgGeneralCompleted)
	})
}
