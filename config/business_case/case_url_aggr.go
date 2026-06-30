package businessCase

import (
	"time"

	statecell "github.com/arcology-network/common-lib/crdt/statecell"
	"github.com/arcology-network/common-lib/exp/slice"
	"github.com/arcology-network/common-lib/types"
	eushared "github.com/arcology-network/eu/shared"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type UrlAggrSelector struct {
	importMsg              string
	commitMsg              string
	generationCompletedMsg string
	finalizeMsg            string

	apcHandleName             string
	outDBMsg                  string
	outGenerationCompletedMsg string
	outPrecommitMsg           string
	outCommitMsg              string

	dataMsg      string
	listMsg      string
	clearMsg     string
	selectResult string

	msgs     []string
	basepath string
}

func NewUrlAggrSelector(basepath, importMsg, commitMsg, generationCompletedMsg, finalizeMsg, apcHandleName, outDBMsg, outGenerationCompletedMsg, outPrecommitMsg, outCommitMsg, dataMsg, listMsg, clearMsg, selectResult string) *UrlAggrSelector {
	handler := &UrlAggrSelector{
		basepath:                  basepath,
		importMsg:                 importMsg,
		commitMsg:                 commitMsg,
		generationCompletedMsg:    generationCompletedMsg,
		finalizeMsg:               finalizeMsg,
		apcHandleName:             apcHandleName,
		outDBMsg:                  outDBMsg,
		outGenerationCompletedMsg: outGenerationCompletedMsg,
		outPrecommitMsg:           outPrecommitMsg,
		outCommitMsg:              outCommitMsg,
		dataMsg:                   dataMsg,
		listMsg:                   listMsg,
		clearMsg:                  clearMsg,
		selectResult:              selectResult,

		msgs: []string{},
	}
	return handler
}

func (ua *UrlAggrSelector) Inputs() ([]string, bool) {
	return []string{
		ua.apcHandleName,
		ua.outDBMsg,
		ua.outGenerationCompletedMsg,
		ua.outPrecommitMsg,
		ua.outCommitMsg,
		ua.selectResult,
		scommon.MsgUrlUpdate,
		scommon.MsgObjectCached,
	}, false
}

func (ua *UrlAggrSelector) Outputs() map[string]int {
	return map[string]int{
		ua.importMsg:              1,
		ua.commitMsg:              1,
		ua.generationCompletedMsg: 1,
		ua.finalizeMsg:            1,
		ua.dataMsg:                1,
		ua.listMsg:                1,
		ua.clearMsg:               1,
		scommon.MsgInitialization: 1,
	}
}

func (ua *UrlAggrSelector) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(ua.apcHandleName, ua.receivedMsgs)
	reg.Register(ua.outDBMsg, ua.receivedMsgs)
	reg.Register(ua.outGenerationCompletedMsg, ua.receivedMsgs)
	reg.Register(ua.outPrecommitMsg, ua.receivedMsgs)
	reg.Register(ua.outCommitMsg, ua.receivedMsgs)
	reg.Register(ua.selectResult, ua.receivedMsgs)
	reg.Register(scommon.MsgUrlUpdate, ua.receivedMsgs)
	reg.Register(scommon.MsgObjectCached, ua.receivedMsgs)
}

func (ua *UrlAggrSelector) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	ua.msgs = append(ua.msgs, msg.Name)
	return nil
}

func (ua *UrlAggrSelector) startTest(ss *broker.StatefulStreamer) []string {
	_, store, univalues := MakeStateStore(ua.basepath)

	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		Store: store,
	})
	m.Height = 1
	ss.Send(scommon.MsgInitialization, m)

	hashs := []evmCommon.Hash{
		evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
		evmCommon.BytesToHash([]byte{7, 8, 9, 10, 11, 12}),
		evmCommon.BytesToHash([]byte{13, 14, 15, 16, 17, 18}),
		evmCommon.BytesToHash([]byte{19, 20, 21, 22, 23, 24}),
	}

	sendingEuResults := []*eushared.EuResult{
		&eushared.EuResult{
			Hash:    hashs[0],
			ID:      uint64(10),
			Status:  uint64(1),
			GasUsed: uint64(100),
			Trans:   univalues[0:2],
		},
		&eushared.EuResult{
			Hash:    hashs[1],
			ID:      uint64(11),
			Status:  uint64(1),
			GasUsed: uint64(110),
			Trans:   univalues[2:4],
		},
	}

	if ua.importMsg == scommon.MsgEuResults {
		euresults := eushared.Euresults(sendingEuResults)
		m = scommon.NewMessageForStream(ua.importMsg, &euresults)
		m.Height = 1
		ss.Send(ua.importMsg, m)
	}
	if ua.importMsg == scommon.MsgNonceEuResults {
		for i := range sendingEuResults {
			nonceTransactions := slice.CloneIf(sendingEuResults[i].Trans, func(v *statecell.StateCell) bool {
				path := *v.GetPath()
				return path[len(path)-5:] == "nonce"
			}, func(v *statecell.StateCell) *statecell.StateCell {
				return v.Clone().(*statecell.StateCell)
			})
			sendingEuResults[i].Trans = nonceTransactions
		}
		euresults := eushared.Euresults(sendingEuResults)
		m = scommon.NewMessageForStream(ua.importMsg, &euresults)
		m.Height = 1
		ss.Send(ua.importMsg, m)
	}

	list := &types.InclusiveList{
		HashList:          hashs[0:2],
		Successful:        []bool{true, true},
		NextGenerationIdx: 0,
	}
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgConflictTransitions, []*statecell.StateCell{})
	m.Height = 1
	ss.Send(scommon.MsgConflictTransitions, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(ua.listMsg, list)
	m.Height = 1
	ss.Send(ua.listMsg, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(ua.generationCompletedMsg, 1)
	m.Height = 1
	ss.Send(ua.generationCompletedMsg, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(ua.finalizeMsg, "")
	m.Height = 1
	ss.Send(ua.finalizeMsg, m)
	time.Sleep(1 * time.Second)

	return ua.msgs
}
