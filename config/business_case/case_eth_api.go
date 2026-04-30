package businessCase

import (
	"fmt"

	"time"

	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	jsonrpc "github.com/deliveroo/jsonrpc-go"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type EthApiTest struct {
	basePath string
	msgs     []string
	sender   actor.OutboundSender
	from     string
}

func NewEthApiTest(basePath string) *EthApiTest {
	handler := &EthApiTest{
		basePath: basePath,
		msgs:     []string{},
		from:     "ethapiTest",
	}
	return handler
}

func (et *EthApiTest) Inputs() ([]string, bool) {
	return []string{}, false
}

func (et *EthApiTest) Outputs() map[string]int {
	return map[string]int{}
}

func (et *EthApiTest) SetSender(sender actor.OutboundSender) {
	et.sender = sender
}

func (et *EthApiTest) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgAcctHash, et.receivedMsgs)
}

func (et *EthApiTest) receivedMsgs(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("Received Mag*******", logger.F("name", msg.Name))
	et.msgs = append(et.msgs, msg.Name)
	return nil
}

func (et *EthApiTest) startTest(ss *broker.StatefulStreamer) []string {
	genesis, store, _ := MakeStateStore(et.basePath)
	blockStart := GetBlockStart(genesis)
	currentinfo := GetParentInfo()
	m := scommon.NewMessageForStream(scommon.MsgInitialization, &mtypes.Initialization{
		Store:             store,
		BlockStart:        blockStart,
		ParentInformation: currentinfo,
	})
	m.Height = 10
	ss.Send(scommon.MsgInitialization, m)
	time.Sleep(1 * time.Second)

	client := jsonrpc.NewClient("http://localhost:7545")
	params := []interface{}{}
	var coinbase string
	// eth_coinbase
	err := client.Call("eth_coinbase", params, &coinbase)
	if err != nil {
		fmt.Printf("******eth_coinbase err:%v\n", err)
		return et.msgs
	}
	et.msgs = append(et.msgs, "eth_coinbase")
	fmt.Printf("******eth_coinbase coinbase:%v\n", coinbase)

	//-------------------------------------------
	params = []interface{}{
		map[string]interface{}{
			"fromBlock": "5",
			"toBlock":   "15",
		},
	}

	// var logs []*evmTypes.Log
	var id string
	err = client.Call("eth_newFilter", params, &id)
	if err != nil {
		fmt.Printf("******eth_newFilter err:%v\n", err)
		return et.msgs
	}
	et.msgs = append(et.msgs, "eth_newFilter")
	fmt.Printf("******eth_newFilter Id:%v\n", id)

	mb, tashes := MakeMonacoBlock()
	mblock := BlockWithHeader(mb, genesis)
	receipts := MakeReceipts(tashes)
	m = scommon.NewMessageForStream(scommon.MsgSelectedReceipts, receipts)
	m.Height = 10
	ss.Send(scommon.MsgSelectedReceipts, m)
	time.Sleep(1 * time.Second)

	m = scommon.NewMessageForStream(scommon.MsgPendingBlock, mblock)
	m.Height = 10
	ss.Send(scommon.MsgPendingBlock, m)
	time.Sleep(1 * time.Second)

	time.Sleep(5 * time.Second)

	params = []interface{}{
		id,
	}

	var logs interface{}

	err = client.Call("eth_getFilterChanges", params, &logs)
	if err != nil {
		fmt.Printf("******eth_getFilterChanges err:%v\n", err)
		return et.msgs
	}
	et.msgs = append(et.msgs, "eth_getFilterChanges")
	fmt.Printf("******eth_getFilterChanges logs:%v\n", logs)

	//----------------------*******************************************\
	m = scommon.NewMessageForStream(scommon.MsgApcHandle, store)
	m.Height = 10
	ss.Send(scommon.MsgApcHandle, m)
	time.Sleep(1 * time.Second)

	//------------------------
	mmb, tashes := MakeMonacoBlockFromGenesis(genesis)
	blockHash := evmCommon.BytesToHash(mmb.Blockhash)
	_, err = et.sender.SendSync("blockstore", "Save", mmb, 10, et.from)
	if err != nil {
		fmt.Printf("******blockstore.Save err:%v\n", err)
		return et.msgs
	}
	et.msgs = append(et.msgs, "blockstore.Save")

	keys := make([]string, len(tashes))
	for i := range keys {
		keys[i] = string(tashes[i].Bytes())
	}

	_, err = et.sender.SendSync("indexerstore", "Save", &storage.SaveIndexRequest{
		Height: mmb.Height,
		Keys:   keys,
		Hash:   string(blockHash.Bytes()),
		IsSave: true,
	}, 10, et.from)
	if err != nil {
		fmt.Printf("******indexerstore.Save err:%v\n", err)
		return et.msgs
	}
	et.msgs = append(et.msgs, "indexerstore.Save")

	//---------------------------------------------------------------
	// params = []interface{}{
	// 	"0x21522c86A586e696961b68aa39632948D9F11170",
	// 	[]interface{}{},
	// 	hexutil.Encode(blockHash.Bytes()),
	// }
	// var proof interface{}

	// err = client.Call("eth_getProof", params, &proof)
	// if err != nil {
	// 	fmt.Printf("******eth_getProof err:%v\n", err)
	// 	return et.msgs
	// }
	// et.msgs = append(et.msgs, "eth_getProof")
	// fmt.Printf("******eth_getProof logs:%v\n", logs)

	return et.msgs
}
