package storage

import (
	"fmt"
	"time"

	ccurl "github.com/arcology-network/storage-committer"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"

	eushared "github.com/arcology-network/eu/shared"
	"github.com/arcology-network/storage-committer/interfaces"
	ccdb "github.com/arcology-network/storage-committer/storage"
	univaluepk "github.com/arcology-network/storage-committer/univalue"
)

type DBOperation interface {
	Init(db interfaces.Datastore, stateCommitter *ccurl.StateCommitter, broker *actor.MessageWrapper)
	Import(transitions []*univaluepk.Univalue)
	PostImport(euResults []*eushared.EuResult, height uint64)
	PreCommit(euResults []*eushared.EuResult, height uint64)
	PostCommit(euResults []*eushared.EuResult, height uint64)
	Finalize()
	Outputs() map[string]int
	Config(params map[string]interface{})
}

type BasicDBOperation struct {
	DB             interfaces.Datastore
	StateCommitter *ccurl.StateCommitter
	MsgBroker      *actor.MessageWrapper

	stateRoot [32]byte

	Keys   []string
	Values []interface{}
}

func (op *BasicDBOperation) Init(db interfaces.Datastore, stateCommitter *ccurl.StateCommitter, broker *actor.MessageWrapper) {
	op.DB = db
	op.StateCommitter = stateCommitter
	op.MsgBroker = broker
	op.Keys = []string{}
	op.Values = []interface{}{}
}

func (op *BasicDBOperation) Import(transitions []*univaluepk.Univalue) {
	op.StateCommitter.Import(transitions)
}

func (op *BasicDBOperation) PostImport(euResults []*eushared.EuResult, height uint64) {
	if height == 0 {
		_, transitions := GetTransitions(euResults)
		op.StateCommitter.Import(transitions)
	}
	op.StateCommitter.Sort()
}

func (op *BasicDBOperation) PreCommit(euResults []*eushared.EuResult, height uint64) {
	//op.StateCommitter.Finalize(GetTransitionIds(euResults))
	op.stateRoot = op.StateCommitter.Precommit(GetTransitionIds(euResults))
	op.Keys = []string{}
	op.Values = []interface{}{}
}

func (op *BasicDBOperation) PostCommit(euResults []*eushared.EuResult, height uint64) {
	// op.stateRoot, op.Keys, op.Values = op.StateCommitter.CopyToDbBuffer()
}

func (op *BasicDBOperation) Finalize() {
	op.StateCommitter.Commit()
}

func (op *BasicDBOperation) Outputs() map[string]int {
	return map[string]int{}
}

func (op *BasicDBOperation) Config(params map[string]interface{}) {}

const (
	dbStateUninit = iota
	dbStateInit
	dbStateDone
)

type DBHandler struct {
	actor.WorkerThread

	db             interfaces.Datastore
	stateCommitter *ccurl.StateCommitter
	state          int
	commitMsg      string
	finalizeMsg    string
	op             DBOperation
}

func NewDBHandler(concurrency int, groupId string, commitMsg, finalizeMsg string, op DBOperation) *DBHandler {
	handler := &DBHandler{
		state:       dbStateUninit,
		commitMsg:   commitMsg,
		finalizeMsg: finalizeMsg,
		op:          op,
	}
	handler.Set(concurrency, groupId)
	return handler
}

func (handler *DBHandler) Inputs() ([]string, bool) {
	msgs := []string{actor.MsgEuResults, handler.commitMsg, handler.finalizeMsg}
	if handler.state == dbStateUninit {
		msgs = append(msgs, actor.MsgInitDB)
	}
	return msgs, false
}

func (handler *DBHandler) Outputs() map[string]int {
	return handler.op.Outputs()
}

func (handler *DBHandler) Config(params map[string]interface{}) {
	if v, ok := params["init_db"]; !ok {
		panic("parameter not found: init_db")
	} else {
		if !v.(bool) {

			handler.db = ccdb.NewParallelEthMemDataStore()

			handler.stateCommitter = ccurl.NewStorageCommitter(handler.db)

			handler.op.Init(handler.db, handler.stateCommitter, handler.MsgBroker)
			handler.state = dbStateDone
		}
	}
	handler.op.Config(params)
}

func (handler *DBHandler) OnStart() {
	handler.op.Init(handler.db, handler.stateCommitter, handler.MsgBroker)
}

func (handler *DBHandler) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch handler.state {
	case dbStateUninit:
		if msg.Name == actor.MsgInitDB {
			handler.db = msg.Data.(interfaces.Datastore)
			handler.stateCommitter = ccurl.NewStorageCommitter(handler.db)

			handler.op.Init(handler.db, handler.stateCommitter, handler.MsgBroker)
			handler.state = dbStateDone
		}
	case dbStateInit:
		if msg.Name == actor.MsgEuResults {
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

			handler.AddLog(log.LogLevel_Info, "Before PostImport.")
			handler.op.PostImport(data, msg.Height)
			handler.AddLog(log.LogLevel_Info, "Before PreCommit.")
			handler.op.PreCommit(data, msg.Height)
			handler.AddLog(log.LogLevel_Info, "Before PostCommit.")
			handler.op.PostCommit(data, msg.Height)
			handler.AddLog(log.LogLevel_Info, "After PostCommit.")
			handler.state = dbStateDone
		}
	case dbStateDone:
		if msg.Name == handler.finalizeMsg {
			handler.AddLog(log.LogLevel_Info, "Before Finalize.")
			handler.op.Finalize()
			handler.AddLog(log.LogLevel_Info, "After Finalize.")
			handler.state = dbStateInit
		}
	}
	return nil
}

func (handler *DBHandler) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		dbStateUninit: {actor.MsgInitDB},
		dbStateInit:   {actor.MsgEuResults, handler.commitMsg},
		dbStateDone:   {actor.MsgEuResults, handler.finalizeMsg},
	}
}

func (handler *DBHandler) GetCurrentState() int {
	return handler.state
}
