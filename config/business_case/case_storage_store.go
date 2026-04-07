package businessCase

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/storage/transactional"
	"github.com/arcology-network/consensus-engine/crypto/tmhash"
	tmproto "github.com/arcology-network/consensus-engine/proto/tendermint/types"
	contyp "github.com/arcology-network/consensus-engine/types"
	conwrk "github.com/arcology-network/main/modules/consensus"
	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/broker"
	scommon "github.com/arcology-network/streamer/common"

	abci "github.com/arcology-network/consensus-engine/abci/types"
	cfg "github.com/arcology-network/consensus-engine/config"
	tmstate "github.com/arcology-network/consensus-engine/proto/tendermint/state"
	sm "github.com/arcology-network/consensus-engine/state"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	dbm "github.com/tendermint/tm-db"
)

type StorageStore struct {
	msgs     []string
	sender   actor.OutboundSender
	from     string
	basePath string
}

func NewStorageStore(basePath string) *StorageStore {
	handler := &StorageStore{
		msgs:     []string{},
		from:     "storageStoreTest",
		basePath: basePath,
	}
	return handler
}

func (ss *StorageStore) Inputs() ([]string, bool) {
	return []string{
		"multiAddress",
	}, false
}

func (ss *StorageStore) Outputs() map[string]int {
	return map[string]int{
		"multiAddress": 1,
	}
}

func (ss *StorageStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("multiAddress", ss.multiAddress)
	reg.Register("multiAddressResult", ss.multiAddressResult)
}
func (ss *StorageStore) SetSender(sender actor.OutboundSender) {
	ss.sender = sender
}

func (ss *StorageStore) multiAddress(ctx *actor.ActionContext) error {
	addrs := []string{
		"0xaB01a3BfC5de6b5Fc481e18F274ADBdbA9B111f0",
		"0x21522c86A586e696961b68aa39632948D9F11170",
		"0xa75Cd05BF16BbeA1759DE2A66c0472131BC5Bd8D",
		"0x2c7161284197e40E83B1b657e98B3bb8FF3C90ed",
		"0x57170608aE58b7d62dCdC3cbDb564C05dDBB7eee",
		"0x9F79316c20f3F83Fcf43deE8a1CeA185A47A5c45",
		"0x9f9E0F23aFd5404b34006678c900629183c9A25d",
		"0xd7cB260c7658589fe68789F2d678e1e85F7e4831",
		"0x230DCCC4660dcBeCb8A6AEA1C713eE7A04B35cAD",
		"0x8aa62d370585e28fd2333325d3dbaef6112279Ce",
	}
	for i := range addrs {
		ctx.ExecCtx.InvokeRPC("urlstore", "GetBalance", addrs[i], "multiAddressResult")
	}
	return nil
}

func (ss *StorageStore) multiAddressResult(ctx *actor.ActionContext) error {
	balance := ctx.RPC.Request.(*big.Int)
	fmt.Printf("StorageStore.multiAddressResult ret:=%v\n", balance)
	return nil
}

func (ss *StorageStore) startTestTransactional(broker *broker.StatefulStreamer) []string {
	// var na int
	txID := fmt.Sprintf("%d", 1)
	_, err := ss.sender.SendSync("transactionalstore", "BeginTransaction", txID, 1, ss.from)
	if err != nil {
		fmt.Printf("******transactionalstore.BeginTransaction err:%v\n", err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, "transactionalstore.BeginTransaction")

	//
	_, err = ss.sender.SendSync("transactionalstore", "AddData", &transactional.AddDataRequest{
		Data:        []byte{0, 1, 2},
		RecoverFunc: "TestByte",
	}, 1, ss.from)
	if err != nil {
		fmt.Printf("******transactionalstore.AddData err:%v\n", err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, "transactionalstore.AddData")

	_, err = ss.sender.SendSync("transactionalstore", "Recover", txID, 1, ss.from)
	if err != nil {
		fmt.Printf("******transactionalstore.Recover err:%v\n", err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, "transactionalstore.Recover")

	_, err = ss.sender.SendSync("transactionalstore", "EndTransaction", "", 1, ss.from)
	if err != nil {
		fmt.Printf("******transactionalstore.EndTransaction err:%v\n", err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, "transactionalstore.EndTransaction")

	return ss.msgs
}

func (ss *StorageStore) startTestUrlStoreMuitiAddress(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}
	_, store, _ := MakeStateStore(ss.basePath)

	_, err := ss.sender.SendSync("urlstore", "Init", store.ReadOnlyStore(), 1, ss.from)
	if err != nil {
		fmt.Printf("******urlstore.Init err:%v\n", err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, "urlstore.Init")

	m := scommon.NewMessageForStream("multiAddress", "")
	m.Height = 10
	broker.Send("multiAddress", m)
	time.Sleep(5 * time.Second)

	return ss.msgs
}

func (ss *StorageStore) startTestUrlStore(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}
	_, store, _ := MakeStateStore(ss.basePath)

	_, err := ss.sender.SendSync("urlstore", "Init", store.ReadOnlyStore(), 1, ss.from)
	if err != nil {
		fmt.Printf("******urlstore.Init err:%v\n", err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, "urlstore.Init")

	addr := "0xa75Cd05BF16BbeA1759DE2A66c0472131BC5Bd8D"

	ret, err := ss.sender.SendSync("urlstore", "GetBalance", addr, 1, ss.from)
	if err != nil || ret == nil {
		fmt.Printf("******urlstore.GetBalance err:%v\n", err)
		return ss.msgs
	}
	bal := ret.(*big.Int)
	fmt.Printf("******urlstore.GetBalance val:%v\n", bal)
	ss.msgs = append(ss.msgs, "urlstore.GetBalance")

	ret, err = ss.sender.SendSync("urlstore", "GetNonce", addr, 1, ss.from)
	if err != nil || ret == nil {
		fmt.Printf("******urlstore.GetNonce err:%v\n", err)
		return ss.msgs
	}
	nonce := ret.(uint64)
	fmt.Printf("******urlstore.GetNonce val:%v\n", nonce)
	ss.msgs = append(ss.msgs, "urlstore.GetNonce")

	ret, err = ss.sender.SendSync("urlstore", "GetCode", addr, 1, ss.from)
	if err != nil || ret == nil {
		fmt.Printf("******urlstore.GetCode err:%v\n", err)
		return ss.msgs
	}
	code := ret.([]byte)
	fmt.Printf("******urlstore.GetCode val:%x\n", code)
	ss.msgs = append(ss.msgs, "urlstore.GetCode")

	ret, err = ss.sender.SendSync("urlstore", "GetEthStorage", &mtypes.UrlEthStorageGetRequest{
		Address: addr,
		Key:     "0x0000000000000000000000000000000000000000000000000000000000000000",
	}, 1, ss.from)
	if err != nil || ret == nil {
		fmt.Printf("******urlstore.GetEthStorage err:%v\n", err)
		return ss.msgs
	}
	storageCode := ret.([]byte)
	fmt.Printf("******urlstore.GetEthStorage val:%x\n", storageCode)
	ss.msgs = append(ss.msgs, "urlstore.GetEthStorage")

	return ss.msgs
}

func (ss *StorageStore) startTestBlockStore(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}
	sername := "blockstore"

	//--------------------
	mblock, _ := MakeMonacoBlock()
	_, err := ss.sender.SendSync(sername, "Save", mblock, 10, ss.from)
	if err != nil {
		fmt.Printf("******%v.Save err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".Save")

	//--------------------
	b, err := ss.sender.SendSync(sername, "GetByHeight", uint64(10), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetByHeight err:%v\n", sername, err)
		return ss.msgs
	}
	block := b.(*mtypes.MonacoBlock)
	fmt.Printf("******%v.GetByHeight val:%v\n", sername, block)
	ss.msgs = append(ss.msgs, sername+".GetByHeight")

	//--------------------
	t, err := ss.sender.SendSync(sername, "GetTransaction", &mstypes.Position{
		Height:     10,
		IdxInBlock: 1,
	}, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetTransaction err:%v\n", sername, err)
		return ss.msgs
	}
	tx := t.(*evmTypes.Transaction)
	fmt.Printf("******%v.GetTransaction val:%v\n", sername, tx)
	ss.msgs = append(ss.msgs, sername+".GetTransaction")

	return ss.msgs
}

func (ss *StorageStore) startTestIndexStore(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}
	sername := "indexerstore"

	//--------------------
	mblock, txhashes := MakeMonacoBlock()
	blockHeight := big.NewInt(10).SetUint64(mblock.Height)
	keys := make([]string, len(txhashes))
	for i := range keys {
		keys[i] = string(txhashes[i].Bytes())
	}
	blockHash := evmCommon.BytesToHash([]byte{101, 102, 103, 104, 105, 106, 107, 108})
	_, err := ss.sender.SendSync(sername, "Save", &storage.SaveIndexRequest{
		Height: 10,
		Keys:   keys,
		Hash:   string(blockHash.Bytes()),
		IsSave: true,
	}, 10, ss.from)
	if err != nil {
		fmt.Printf("******%v.Save err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".Save")

	//--------------------
	h, err := ss.sender.SendSync(sername, "GetHeightByHash", string(blockHash.Bytes()), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetHeightByHash err:%v\n", sername, err)
		return ss.msgs
	}
	height := h.(*big.Int)
	fmt.Printf("******%v.GetHeightByHash val:%v\n", sername, height)
	ss.msgs = append(ss.msgs, sername+".GetHeightByHash")

	//--------------------
	p, err := ss.sender.SendSync(sername, "GetPosition", keys[1], 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetPosition err:%v\n", sername, err)
		return ss.msgs
	}
	position := p.(*mstypes.Position)
	fmt.Printf("******%v.GetPosition val:%v\n", sername, position)
	ss.msgs = append(ss.msgs, sername+".GetPosition")

	//--------------------
	hs, err := ss.sender.SendSync(sername, "GetBlockHashes", blockHeight.Uint64(), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetBlockHashes err:%v\n", sername, err)
		return ss.msgs
	}
	hss := hs.([]string)
	fmt.Printf("******%v.GetBlockHashes val:%v\n", sername, hss)
	ss.msgs = append(ss.msgs, sername+".GetBlockHashes")

	return ss.msgs
}

func (ss *StorageStore) startTestReceiptStore(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}
	sername := "receiptstore"

	blockHeight := big.NewInt(10)
	//--------------------
	_, txhashes := MakeMonacoBlock()
	receipts := MakeReceipts(txhashes)
	_, err := ss.sender.SendSync(sername, "Save", &storage.SaveReceiptsRequest{
		Height:   10,
		Receipts: receipts,
	}, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Save err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".Save")

	//--------------------
	r, err := ss.sender.SendSync(sername, "Get", &mstypes.Position{
		Height:     blockHeight.Uint64(),
		IdxInBlock: 1,
	}, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Get err:%v\n", sername, err)
		return ss.msgs
	}
	receipt := r.(*evmTypes.Receipt)
	fmt.Printf("******%v.Get val:%v\n", sername, receipt)
	ss.msgs = append(ss.msgs, sername+".Get")

	//--------------------
	rs, err := ss.sender.SendSync(sername, "GetBlockReceipts", blockHeight.Uint64(), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetBlockReceipts err:%v\n", sername, err)
		return ss.msgs
	}
	receipts = rs.([]*evmTypes.Receipt)
	fmt.Printf("******%v.GetBlockReceipts val:%v\n", sername, receipts)
	ss.msgs = append(ss.msgs, sername+".GetBlockReceipts")

	return ss.msgs
}

// func (ss *StorageStore) startTestSchdStore(broker *broker.StatefulStreamer) []string {
// 	ss.msgs = []string{}
// 	sername := "schdstore"

// 	//-----------------
// 	txID := fmt.Sprintf("%d", 10)
// 	_, err := ss.sender.SendSync("transactionalstore", "BeginTransaction", txID, 1, ss.from)
// 	if err != nil {
// 		fmt.Printf("******transactionalstore.BeginTransaction err:%v\n", err)
// 		return ss.msgs
// 	}
// 	ss.msgs = append(ss.msgs, "transactionalstore.BeginTransaction")

//--------------------
// schdState := &mtypes.SchdState{
// 	Height:                10,
// 	NewContracts:          []evmCommon.Address{evmCommon.BytesToAddress([]byte{1, 2, 3, 4, 5, 6})},
// 	ConflictionLefts:      []evmCommon.Address{evmCommon.BytesToAddress([]byte{11, 12, 13, 14, 15, 16})},
// 	ConflictionRights:     []evmCommon.Address{evmCommon.BytesToAddress([]byte{21, 22, 23, 24, 25, 26})},
// 	ConflictionLeftSigns:  [][4]byte{[4]byte{1, 2, 3, 4}},
// 	ConflictionRightSigns: [][4]byte{[4]byte{15, 12, 13, 14}},
// }

// _, err = ss.sender.SendSync(sername, "Save", schdState, 3, ss.from)
// if err != nil {
// 	fmt.Printf("******%v.Save err:%v\n", sername, err)
// 	return ss.msgs
// }
// ss.msgs = append(ss.msgs, sername+".Save")

//--------------------
// schdState.NewContracts = []evmCommon.Address{evmCommon.BytesToAddress([]byte{11, 2, 3, 4, 5, 6})}
// _, err = ss.sender.SendSync(sername, "DirectWrite", schdState, 3, ss.from)
// if err != nil {
// 	fmt.Printf("******%v.DirectWrite err:%v\n", sername, err)
// 	return ss.msgs
// }
// ss.msgs = append(ss.msgs, sername+".DirectWrite")

//--------------------
// 	state, err := ss.sender.SendSync(sername, "Load", "", 3, ss.from)
// 	if err != nil {
// 		fmt.Printf("******%v.Load err:%v\n", sername, err)
// 		return ss.msgs
// 	}
// 	states := state.([]mtypes.SchdState)
// 	fmt.Printf("******%v.Load val:%x\n", sername, states)
// 	ss.msgs = append(ss.msgs, sername+".Load")

// 	//-------------------------
// 	_, err = ss.sender.SendSync("transactionalstore", "EndTransaction", "", 1, ss.from)
// 	if err != nil {
// 		fmt.Printf("******transactionalstore.EndTransaction err:%v\n", err)
// 		return ss.msgs
// 	}
// 	ss.msgs = append(ss.msgs, "transactionalstore.EndTransaction")

// 	return ss.msgs
// }

func (ss *StorageStore) startTestStatestore(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}
	sername := "statestore"

	//--------------------
	curState := storage.State{
		Height:        10,
		ParentHash:    evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
		ParentRoot:    evmCommon.BytesToHash([]byte{11, 12, 13, 14, 15, 16}),
		ExcessBlobGas: uint64(234567),
		BlobGasUsed:   uint64(3546789),
	}
	_, err := ss.sender.SendSync(sername, "Save", &curState, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Save err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".Save")

	//--------------------
	height, err := ss.sender.SendSync(sername, "GetHeight", "", 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetHeight err:%v\n", sername, err)
		return ss.msgs
	}
	uheight := height.(uint64)
	fmt.Printf("******%v.GetHeight val:%x\n", sername, uheight)
	ss.msgs = append(ss.msgs, sername+".GetHeight")

	//--------------------
	p, err := ss.sender.SendSync(sername, "GetParentInfo", "", 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.GetParentInfo err:%v\n", sername, err)
		return ss.msgs
	}
	parentInfo := p.(*mtypes.ParentInfo)
	fmt.Printf("******%v.GetParentInfo val:%x\n", sername, parentInfo)
	ss.msgs = append(ss.msgs, sername+".GetParentInfo")

	return ss.msgs
}

func (ss *StorageStore) startTestTmStatestore(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}
	sername := "tmstatestore"

	//--------------------
	doc, err := contyp.GenesisDocFromFile("./genesis-consensus.json")
	if err != nil {
		fmt.Printf("******%v.GenesisDocFromFile err:%v\n", sername, err)
		return ss.msgs
	}
	state, err := ss.sender.SendSync(sername, "LoadFromDBOrGenesisDoc", doc, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadFromDBOrGenesisDoc err:%v\n", sername, err)
		return ss.msgs
	}
	smState := state.(sm.State)
	fmt.Printf("******%v.LoadFromDBOrGenesisDoc val:%x\n", sername, smState)
	ss.msgs = append(ss.msgs, sername+".LoadFromDBOrGenesisDoc")

	//------------------------
	_, err = ss.sender.SendSync(sername, "Save", &smState, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Save err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".Save")

	//------------------------
	state, err = ss.sender.SendSync(sername, "Load", "", 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Load err:%v\n", sername, err)
		return ss.msgs
	}
	smState = state.(sm.State)
	fmt.Printf("******%v.Load val:%x\n", sername, smState)
	ss.msgs = append(ss.msgs, sername+".Load")

	//------------------------
	_, err = ss.sender.SendSync(sername, "Bootstrap", &smState, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Bootstrap err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".Bootstrap")

	//------------------------
	vs, err := ss.sender.SendSync(sername, "LoadValidators", int64(10), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadValidators err:%v\n", sername, err)
		return ss.msgs
	}
	vss := vs.(*contyp.ValidatorSet)
	fmt.Printf("******%v.LoadValidators val:%x\n", sername, vss)
	ss.msgs = append(ss.msgs, sername+".LoadValidators")

	//------------------------
	cp, err := ss.sender.SendSync(sername, "LoadConsensusParams", int64(10), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadConsensusParams err:%v\n", sername, err)
		return ss.msgs
	}
	cps := cp.(tmproto.ConsensusParams)
	fmt.Printf("******%v.LoadConsensusParams val:%x\n", sername, cps)
	ss.msgs = append(ss.msgs, sername+".LoadConsensusParams")

	//------------------------
	responses := &tmstate.ABCIResponses{
		DeliverTxs: []*abci.ResponseDeliverTx{
			{Code: 0, Data: []byte{0x01}, Log: "ok"},
			{Code: 0, Data: []byte{0x02}, Log: "ok"},
			{Code: 1, Log: "not ok"},
		},
		EndBlock:   &abci.ResponseEndBlock{},
		BeginBlock: &abci.ResponseBeginBlock{},
	}
	_, err = ss.sender.SendSync(sername, "SaveABCIResponses", &conwrk.SaveABCIResponsesRequest{
		Height:        int64(10),
		ABCIResponses: responses,
	}, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.SaveABCIResponses err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".SaveABCIResponses")

	//------------------------
	resp, err := ss.sender.SendSync(sername, "LoadABCIResponses", int64(10), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadABCIResponses err:%v\n", sername, err)
		return ss.msgs
	}
	resps := resp.(*tmstate.ABCIResponses)
	fmt.Printf("******%v.LoadABCIResponses val:%x\n", sername, resps)
	ss.msgs = append(ss.msgs, sername+".LoadABCIResponses")

	//------------------------
	_, err = ss.sender.SendSync(sername, "PruneStates", &conwrk.PruneStatesRequest{
		From: 9,
		To:   10,
	}, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.PruneStates err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".PruneStates")

	return ss.msgs
}

func MakeState() *sm.State {
	config := cfg.ResetTestRoot("state_")
	dbType := dbm.BackendType(config.DBBackend)
	stateDB, _ := dbm.NewDB("state", dbType, config.DBDir())
	stateStore := sm.NewStore(stateDB)
	state, _ := stateStore.LoadFromDBOrGenesisFile(config.GenesisFile())
	return &state
}

func (ss *StorageStore) startTestTmBlockstore(broker *broker.StatefulStreamer) []string {
	ss.msgs = []string{}

	sername := "tmblockstore"

	base, err := ss.sender.SendSync(sername, "Base", "", 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Base err:%v\n", sername, err)
		return ss.msgs
	}
	baseInt := base.(int64)
	fmt.Printf("******%v.Base val:%x\n", sername, baseInt)
	ss.msgs = append(ss.msgs, sername+".Base")

	height, err := ss.sender.SendSync(sername, "Height", "", 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Height err:%v\n", sername, err)
		return ss.msgs
	}
	heightInt := height.(int64)
	fmt.Printf("******%v.Height val:%x\n", sername, heightInt)
	ss.msgs = append(ss.msgs, sername+".Height")

	size, err := ss.sender.SendSync(sername, "Size", "", 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.Size err:%v\n", sername, err)
		return ss.msgs
	}
	sizeInt := size.(int64)
	fmt.Printf("******%v.Size val:%x\n", sername, sizeInt)
	ss.msgs = append(ss.msgs, sername+".Size")
	//-----------------------------------
	blockHaight := int64(3)
	block := MakeBlock(blockHaight)
	block.ProposerAddress = contyp.Address{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

	blockParts := block.MakePartSet(500)

	_, err = ss.sender.SendSync(sername, "SaveBlock", &conwrk.SaveBlockRequest{
		Block:      block,
		BlockParts: blockParts,
		SeenCommit: block.LastCommit,
	}, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.SaveBlock err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".SaveBlock")

	loadBaseMeta, err := ss.sender.SendSync(sername, "LoadBaseMeta", "", 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadBaseMeta err:%v\n", sername, err)
		return ss.msgs
	}
	blockMeta := loadBaseMeta.(*contyp.BlockMeta)
	fmt.Printf("******%v.LoadBaseMeta val:%x\n", sername, blockMeta)
	ss.msgs = append(ss.msgs, sername+".LoadBaseMeta")

	loadBlockMeta, err := ss.sender.SendSync(sername, "LoadBlockMeta", blockHaight, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadBlockMeta err:%v\n", sername, err)
		return ss.msgs
	}
	blockMeta = loadBlockMeta.(*contyp.BlockMeta)
	fmt.Printf("******%v.LoadBlockMeta val:%x\n", sername, blockMeta)
	ss.msgs = append(ss.msgs, sername+".LoadBlockMeta")

	retblock, err := ss.sender.SendSync(sername, "LoadBlock", blockHaight, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadBlock err:%v\n", sername, err)
		return ss.msgs
	}
	block1 := retblock.(*contyp.Block)
	fmt.Printf("******%v.LoadBlock val:%x\n", sername, block1)
	ss.msgs = append(ss.msgs, sername+".LoadBlock")

	hblock, err := ss.sender.SendSync(sername, "LoadBlockByHash", block.Hash().Bytes(), 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadBlockByHash err:%v\n", sername, err)
		return ss.msgs
	}
	block1 = hblock.(*contyp.Block)
	fmt.Printf("******%v.LoadBlockByHash val:%v\n", sername, block1)
	ss.msgs = append(ss.msgs, sername+".LoadBlockByHash")

	p, err := ss.sender.SendSync(sername, "LoadBlockPart", &conwrk.LoadBlockPartRequest{
		Height: 3,
		Index:  0,
	}, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadBlockPart err:%v\n", sername, err)
		return ss.msgs
	}
	blockpart := p.(*contyp.Part)
	fmt.Printf("******%v.LoadBlockPart val:%v\n", sername, blockpart)
	ss.msgs = append(ss.msgs, sername+".LoadBlockPart")

	bc, err := ss.sender.SendSync(sername, "LoadBlockCommit", blockHaight, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadBlockCommit err:%v\n", sername, err)
		return ss.msgs
	}
	blockCommit := bc.(*contyp.Commit)
	fmt.Printf("******%v.LoadBlockCommit val:%v\n", sername, blockCommit)
	ss.msgs = append(ss.msgs, sername+".LoadBlockCommit")

	bc, err = ss.sender.SendSync(sername, "LoadSeenCommit", blockHaight, 3, ss.from)
	if err != nil {
		fmt.Printf("******%v.LoadSeenCommit err:%v\n", sername, err)
		return ss.msgs
	}
	blockCommit = bc.(*contyp.Commit)
	fmt.Printf("******%v.LoadSeenCommit val:%v\n", sername, blockCommit)
	ss.msgs = append(ss.msgs, sername+".LoadSeenCommit")

	prune, err := ss.sender.SendSync(sername, "PruneBlocks", blockHaight, 4, ss.from)
	if err != nil {
		fmt.Printf("******%v.PruneBlocks err:%v\n", sername, err)
		return ss.msgs
	}
	pruned := prune.(uint64)
	fmt.Printf("******%v.PruneBlocks val:%v\n", sername, pruned)
	ss.msgs = append(ss.msgs, sername+".PruneBlocks")

	blockHaight = int64(4)
	block = MakeBlock(blockHaight)
	block.ProposerAddress = contyp.Address{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

	_, err = ss.sender.SendSync(sername, "SaveBlockAsync", &conwrk.SaveBlockRequest{
		Block:      block,
		BlockParts: blockParts,
		SeenCommit: block.LastCommit,
	}, 4, ss.from)
	if err != nil {
		fmt.Printf("******%v.SaveBlockAsync err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".SaveBlockAsync")

	_, err = ss.sender.SendSync(sername, "SaveSeenCommit", &conwrk.SaveSeenCommitRequest{
		Height:     4,
		SeenCommit: block.LastCommit,
	}, 4, ss.from)
	if err != nil {
		fmt.Printf("******%v.SaveSeenCommit err:%v\n", sername, err)
		return ss.msgs
	}
	ss.msgs = append(ss.msgs, sername+".SaveSeenCommit")

	return ss.msgs
}

func MakeBlock(h int64) *contyp.Block {
	txs := []contyp.Tx{contyp.Tx("foo"), contyp.Tx("bar")}
	lastID := makeBlockIDRandom()

	voteSet, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, _ := contyp.MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())

	ev := contyp.NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
	evList := []contyp.Evidence{ev}

	return contyp.MakeBlock(h, txs, commit, evList)
}

func randVoteSet(
	height int64,
	round int32,
	signedMsgType tmproto.SignedMsgType,
	numValidators int,
	votingPower int64,
) (*contyp.VoteSet, *contyp.ValidatorSet, []contyp.PrivValidator) {
	valSet, privValidators := contyp.RandValidatorSet(numValidators, votingPower)
	return contyp.NewVoteSet("test_chain_id", height, round, signedMsgType, valSet), valSet, privValidators
}

func makeBlockIDRandom() contyp.BlockID {
	var (
		blockHash   = make([]byte, tmhash.Size)
		partSetHash = make([]byte, tmhash.Size)
	)
	rand.Read(blockHash)   //nolint: errcheck // ignore errcheck for read
	rand.Read(partSetHash) //nolint: errcheck // ignore errcheck for read
	return contyp.BlockID{blockHash, contyp.PartSetHeader{123, partSetHash}}
}
