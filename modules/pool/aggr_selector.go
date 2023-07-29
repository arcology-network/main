package pool

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/ethrpc"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/concurrenturl/interfaces"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/common/hexutil"
	evmTypes "github.com/arcology-network/evm/core/types"
	evmtypes "github.com/arcology-network/evm/core/types"
	evmRlp "github.com/arcology-network/evm/rlp"
	"go.uber.org/zap"
)

type AggrSelector struct {
	actor.WorkerThread

	maxReap      int
	obsoleteTime uint64
	closeCheck   bool
	pool         *Pool
	state        int
	height       uint64
}

const (
	poolStateClean = iota
	poolStateReap
	poolStateCherryPick
)

var (
	rpcInstance actor.IWorkerEx
	initRpcOnce sync.Once
)

// return a Subscriber struct
func NewAggrSelector(concurrency int, groupid string) actor.IWorkerEx {
	// agg := AggrSelector{
	// 	state: poolStateClean,
	// }
	// agg.Set(concurrency, groupid)
	// return &agg

	initRpcOnce.Do(func() {
		rpcInstance = &AggrSelector{
			state: poolStateClean,
		}
		rpcInstance.(*AggrSelector).Set(concurrency, groupid)
	})

	return rpcInstance
}

func (a *AggrSelector) Inputs() ([]string, bool) {
	return []string{
		actor.MsgNonceReady,
		actor.MsgMessager,
		actor.MsgReapCommand,
		actor.MsgReapinglist,
	}, false
}

func (a *AggrSelector) Outputs() map[string]int {
	return map[string]int{
		actor.MsgMessagersReaped: 1,
		actor.MsgMetaBlock:       1,
		actor.MsgSelectedTx:      1,
	}
}

func (a *AggrSelector) Config(params map[string]interface{}) {
	a.maxReap = int(params["max_reap_size"].(float64))
	a.obsoleteTime = uint64(params["obsolete_time"].(float64))
	if _, ok := params["close_check"]; ok {
		a.closeCheck = params["close_check"].(bool)
	}
}

func (a *AggrSelector) OnStart() {
}

func (a *AggrSelector) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch a.state {
	case poolStateClean:
		switch msg.Name {
		case actor.MsgNonceReady:
			if a.pool == nil {
				a.pool = NewPool(*(msg.Data.(*interfaces.Datastore)), a.obsoleteTime, a.closeCheck)
			} else {
				a.pool.Clean(msg.Height)
				a.AddLog(log.LogLevel_Info, fmt.Sprintf("Clear pool on height %d", msg.Height))
			}
			a.state = poolStateReap
			a.height = msg.Height + 1
		}
	case poolStateReap:
		switch msg.Name {
		case actor.MsgMessager:
			msgs := msg.Data.(*types.IncomingMsgs)
			a.pool.Add(msgs.Msgs, msgs.Src, msg.Height)
		case actor.MsgReapCommand:
			reaped := a.pool.Reap(a.maxReap)
			a.send(reaped, true, msg.Height)
			a.state = poolStateCherryPick
			a.AddLog(log.LogLevel_Info, "Reap done, switch to poolStateCherryPick")
		}
	case poolStateCherryPick:
		switch msg.Name {
		case actor.MsgMessager:
			msgs := msg.Data.(*types.IncomingMsgs)
			reaped := a.pool.Add(msgs.Msgs, msgs.Src, msg.Height)
			if reaped != nil {
				a.send(reaped, false, a.height)
				a.state = poolStateClean
				a.AddLog(log.LogLevel_Info, "Data received, switch to poolStateClean")
			}
		case actor.MsgReapinglist:
			a.CheckPoint("pool received reapinglist")
			list := make([]evmCommon.Hash, len(msg.Data.(*types.ReapingList).List))
			for i := range list {
				list[i] = *msg.Data.(*types.ReapingList).List[i]
			}
			reaped := a.pool.CherryPick(list)
			if reaped != nil {
				a.send(reaped, false, msg.Height)
				a.state = poolStateClean
				a.AddLog(log.LogLevel_Info, "List received, switch to poolStateClean")
			}
		}
	}
	return nil
}

func (a *AggrSelector) send(reaped []*types.StandardMessage, isProposer bool, height uint64) {
	a.AddLog(log.LogLevel_Debug, "reap end", zap.Int("reapeds", len(reaped)))
	if isProposer {
		hashes := make([]*evmCommon.Hash, len(reaped))
		for i := range hashes {
			hashes[i] = &reaped[i].TxHash
		}
		a.MsgBroker.Send(actor.MsgMetaBlock, &types.MetaBlock{
			Txs:      [][]byte{},
			Hashlist: hashes,
		}, height)
	} else {
		txs := make([][]byte, len(reaped))
		for i := range txs {
			txs[i] = reaped[i].TxRawData
		}
		// a.MsgBroker.Send(actor.MsgMessagersReaped, types.SendingStandardMessages{
		// 	Data: types.StandardMessages(reaped).EncodeToBytes(),
		// }, height)
		a.MsgBroker.Send(actor.MsgMessagersReaped, reaped, height)
		a.CheckPoint("send messagersReaped")
		a.MsgBroker.Send(actor.MsgSelectedTx, types.Txs{Data: txs}, height)
		a.CheckPoint("send selectedtx")
	}
}

func (a *AggrSelector) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		poolStateClean: {
			actor.MsgNonceReady,
		},
		poolStateReap: {
			actor.MsgMessager,
			actor.MsgReapCommand,
		},
		poolStateCherryPick: {
			actor.MsgMessager,
			actor.MsgReapinglist,
		},
	}
}

func (a *AggrSelector) GetCurrentState() int {
	return a.state
}

func (a *AggrSelector) Height() uint64 {
	if a.height == 0 {
		return math.MaxUint64
	}
	return a.height
}

func (a *AggrSelector) Query(ctx context.Context, request *types.QueryRequest, response *types.QueryResult) error {
	switch request.QueryType {
	case types.QueryType_Transaction:
		hash := request.Data.(evmCommon.Hash)
		st := a.pool.QueryByHash(evmCommon.BytesToHash(hash.Bytes()))
		if st == nil {
			response.Data = nil
			return errors.New("hash not found")
		}
		txReal := st.TxRawData[1:]
		otx := new(evmTypes.Transaction)
		if err := evmRlp.DecodeBytes(txReal, otx); err != nil {
			return errors.New("tx decode err")
		}
		// transactionIndex := uint64(0)
		v, s, r := otx.RawSignatureValues()
		msg := st.Native
		transaction := ethrpc.RPCTransaction{
			// BlockHash:        evmCommon.Hash{},
			// BlockNumber:      big.NewInt(0),
			// TransactionIndex: &transactionIndex,

			Type:     hexutil.Uint64(evmtypes.LegacyTxType),
			From:     evmCommon.Address(msg.From),
			Gas:      hexutil.Uint64(otx.Gas()),
			GasPrice: (*hexutil.Big)(otx.GasPrice()),
			Hash:     hash,
			Input:    hexutil.Bytes(otx.Data()),
			Nonce:    hexutil.Uint64(otx.Nonce()),
			To:       (*evmCommon.Address)(msg.To),
			Value:    (*hexutil.Big)(otx.Value()),
			V:        (*hexutil.Big)(v),
			R:        (*hexutil.Big)(r),
			S:        (*hexutil.Big)(s),
		}

		response.Data = &transaction
	}
	return nil
}
