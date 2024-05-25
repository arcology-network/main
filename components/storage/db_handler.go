package storage

import (
	"fmt"
	"time"

	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"

	eushared "github.com/arcology-network/eu/shared"
	statestore "github.com/arcology-network/storage-committer"
	stgproxy "github.com/arcology-network/storage-committer/storage/proxy"
	univaluepk "github.com/arcology-network/storage-committer/univalue"
)

type DBOperation interface {
	Init(stateStore *statestore.StateStore, broker *actor.MessageWrapper)
	InitAsync()
	Import(transitions []*univaluepk.Univalue)
	PreCommit(euResults []*eushared.EuResult, height uint64)
	PreCommitCompleted()
	Commit(height uint64)
	// ObjectCommit(height uint64)
	Outputs() map[string]int
	Config(params map[string]interface{})
}

type BasicDBOperation struct {
	StateStore *statestore.StateStore
	MsgBroker  *actor.MessageWrapper

	Keys     []string
	Values   []interface{}
	AcctRoot [32]byte
}

func (op *BasicDBOperation) Init(stateStore *statestore.StateStore, broker *actor.MessageWrapper) {
	op.StateStore = stateStore
	op.MsgBroker = broker
	op.Keys = []string{}
	op.Values = []interface{}{}
	op.AcctRoot = [32]byte{}
}

func (op *BasicDBOperation) Import(transitions []*univaluepk.Univalue) {
	fmt.Printf("==================components/storage/db_handler.go  Import transitions size:%v\n", len(transitions))
	op.StateStore.Import(transitions)
}

func (op *BasicDBOperation) PreCommit(euResults []*eushared.EuResult, height uint64) {
	op.StateStore.Finalize(GetTransitionIds(euResults))
	op.StateStore.SyncPrecommit()
	op.Keys = []string{}
	op.Values = []interface{}{}
}

func (op *BasicDBOperation) PreCommitCompleted() {

}
func (op *BasicDBOperation) InitAsync() {

}

func (op *BasicDBOperation) Commit(height uint64) {
	op.StateStore.SyncCommit(height)
}

func (op *BasicDBOperation) Outputs() map[string]int {
	return map[string]int{}
}

func (op *BasicDBOperation) Config(params map[string]interface{}) {}

const (
	dbStateUninit = iota
	dbStateAsyncinit
	dbStateInit
	dbStateDone
)

type DBHandler struct {
	actor.WorkerThread

	StateStore             *statestore.StateStore
	state                  int
	importMsg              string
	commitMsg              string
	generationCompletedMsg string
	finalizeMsg            string
	initDBAsyncMsg         string
	op                     DBOperation
}

func NewDBHandler(concurrency int, groupId string, importMsg, commitMsg, generationCompletedMsg, finalizeMsg, initDBAsyncMsg string, op DBOperation) *DBHandler {
	handler := &DBHandler{
		state:                  dbStateUninit,
		importMsg:              importMsg,
		commitMsg:              commitMsg,
		generationCompletedMsg: generationCompletedMsg,
		finalizeMsg:            finalizeMsg,
		initDBAsyncMsg:         initDBAsyncMsg,
		op:                     op,
	}
	handler.Set(concurrency, groupId)
	return handler
}

func (handler *DBHandler) Inputs() ([]string, bool) {
	msgs := []string{handler.importMsg, handler.commitMsg, handler.generationCompletedMsg, handler.finalizeMsg, handler.initDBAsyncMsg}
	if handler.state == dbStateUninit {
		msgs = append(msgs, actor.MsgInitDB)
	}
	return msgs, false
}

func (handler *DBHandler) Outputs() map[string]int {
	outputs := handler.op.Outputs()
	outputs[handler.initDBAsyncMsg] = 1
	return outputs
}

func (handler *DBHandler) Config(params map[string]interface{}) {
	dbpath := ""
	if v, ok := params["dbpath"]; !ok {
		panic("parameter not found: dbpath")
	} else {
		dbpath = v.(string)
	}
	if v, ok := params["init_db"]; !ok {
		panic("parameter not found: init_db")
	} else {
		if !v.(bool) {

			handler.StateStore = statestore.NewStateStore(stgproxy.NewLevelDBStoreProxy(dbpath))

			handler.op.Init(handler.StateStore, handler.MsgBroker)
			handler.state = dbStateAsyncinit
		}
	}
	handler.op.Config(params)
}

func (handler *DBHandler) OnStart() {
	handler.op.Init(handler.StateStore, handler.MsgBroker)
	// handler.MsgBroker.Send(handler.initDBAsyncMsg, "")
}

func (handler *DBHandler) InitMsgs() []*actor.Message {
	return []*actor.Message{
		{
			Name:   handler.initDBAsyncMsg,
			Height: 0,
		},
	}
}
func (handler *DBHandler) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch handler.state {
	case dbStateUninit:
		if msg.Name == actor.MsgInitDB {
			handler.StateStore = msg.Data.(*statestore.StateStore) //statestore.NewStateStore(handler.db)

			handler.op.Init(handler.StateStore, handler.MsgBroker)
			handler.state = dbStateAsyncinit
			handler.AddLog(log.LogLevel_Debug, ">>>>>change into dbStateDone>>>>>>>>")
		}
	case dbStateAsyncinit:
		if msg.Name == handler.initDBAsyncMsg {
			handler.op.InitAsync()
			handler.state = dbStateDone
		}
	case dbStateInit:
		if msg.Name == handler.importMsg {
			data := msg.Data.(*eushared.Euresults)
			t1 := time.Now()
			_, transitions := GetTransitions(*data)
			handler.op.Import(transitions)
			fmt.Printf("DBHandler Euresults import height:%v,tim:%v\n", msg.Height, time.Since(t1))
		} else if msg.Name == handler.commitMsg {
			var data []*eushared.EuResult
			if msg.Data != nil {
				for _, item := range msg.Data.([]interface{}) {
					data = append(data, item.(*eushared.EuResult))
				}
			}
			if msg.Height == 0 {
				_, transitions := GetTransitions(data)
				handler.op.Import(transitions)
			}
			handler.AddLog(log.LogLevel_Info, "Before PreCommit.")
			handler.op.PreCommit(data, msg.Height)
			handler.AddLog(log.LogLevel_Info, "After PreCommit.")

		} else if msg.Name == handler.generationCompletedMsg {
			handler.op.PreCommitCompleted()
			handler.state = dbStateDone
			handler.AddLog(log.LogLevel_Debug, ">>>>>change into dbStateDone >>>>>>>>")
		}
	case dbStateDone:
		if msg.Name == handler.finalizeMsg {
			handler.AddLog(log.LogLevel_Info, "Before Commit.")
			handler.op.Commit(msg.Height)
			handler.AddLog(log.LogLevel_Info, "After Commit.")
			handler.state = dbStateInit
			handler.AddLog(log.LogLevel_Debug, ">>>>>change into dbStateInit >>>>>>>>")
		}
	}
	return nil
}

func (handler *DBHandler) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		dbStateUninit:    {actor.MsgInitDB},
		dbStateAsyncinit: {handler.initDBAsyncMsg},
		dbStateInit:      {actor.MsgEuResults, handler.commitMsg, handler.generationCompletedMsg},
		dbStateDone:      {actor.MsgEuResults, handler.finalizeMsg},
	}
}

func (handler *DBHandler) GetCurrentState() int {
	return handler.state
}
