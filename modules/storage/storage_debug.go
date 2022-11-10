package storage

import (
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
)

type StorageDebug struct {
	actor.WorkerThread
}

// return a Subscriber struct
func NewStorageDebug(concurrency int, groupid string) actor.IWorkerEx {
	s := StorageDebug{}
	s.Set(concurrency, groupid)
	return &s
}

func (s *StorageDebug) Inputs() ([]string, bool) {
	return []string{actor.MsgExecutingLogs}, false
}

func (s *StorageDebug) Outputs() map[string]int {
	return map[string]int{}
}

func (*StorageDebug) OnStart() {}
func (*StorageDebug) Stop()    {}

func (s *StorageDebug) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgExecutingLogs:
			executingLogs := v.Data.([]string)
			var na int
			intf.Router.Call("debugstore", "SaveLog", &LogSaveRequest{
				Height:   v.Height,
				Execlogs: executingLogs,
			}, &na)
		}
	}
	return nil
}
