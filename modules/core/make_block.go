package core

import (
	"fmt"
	"math"
	"math/big"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/transactional"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
)

type MakeBlock struct {
	actor.WorkerThread
	SignerType uint8
}

// return a Subscriber struct
func NewMakeBlock(concurrency int, groupid string) actor.IWorkerEx {
	in := MakeBlock{}
	in.Set(concurrency, groupid)
	in.SignerType = types.Signer_London
	return &in
}

func (m *MakeBlock) Inputs() ([]string, bool) {
	return []string{
		actor.MsgBlockStart,
		actor.MsgSelectedTx,
		actor.MsgTxHash,
		actor.MsgAcctHash,
		actor.MsgRcptHash,
		actor.MsgGasUsed,
		actor.MsgLocalParentInfo,
		actor.MsgBlockParams,
		actor.MsgBloom,
		actor.MsgWithDrawHash,
	}, true
}

func (m *MakeBlock) Outputs() map[string]int {
	return map[string]int{
		actor.MsgAppHash:         1,
		actor.MsgParentInfo:      1,
		actor.MsgLocalParentInfo: 1,
		actor.MsgPendingBlock:    1,
	}
}

func (m *MakeBlock) OnStart() {
}

func (m *MakeBlock) OnMessageArrived(msgs []*actor.Message) error {

	// coinbase := evmCommon.Address{}
	txhash := evmCommon.Hash{}
	accthash := evmCommon.Hash{}
	rcpthash := evmCommon.Hash{}
	gasused := uint64(0)
	txSelected := [][]byte{}
	parentinfo := &types.ParentInfo{}
	height := uint64(0)
	// timestamp := big.NewInt(0)
	var blockParams *types.BlockParams
	var blockStart *actor.BlockStart
	var bloom evmTypes.Bloom
	var withDrawHash *evmCommon.Hash
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgBlockStart:
			blockStart = v.Data.(*actor.BlockStart)
			// timestamp = bs.Timestamp
			// coinbase = bs.Coinbase
			height = blockStart.Height
		case actor.MsgSelectedTx:
			datas := v.Data.([][]byte)
			isnil, err := m.IsNil(datas, "txSelected")
			if isnil {
				return err
			}
			txSelected = datas // v.Data.([][]byte)
		case actor.MsgTxHash:
			hash := v.Data.(*evmCommon.Hash)
			isnil, err := m.IsNil(hash, "txhash")
			if isnil {
				return err
			}
			txhash = *hash
		case actor.MsgAcctHash:
			hash := v.Data.(*evmCommon.Hash)
			isnil, err := m.IsNil(hash, "accthash")
			if isnil {
				return err
			}
			accthash = *hash
			m.AddLog(log.LogLevel_Info, "received accthash", zap.String("accthash", fmt.Sprintf("%x", accthash)))
		case actor.MsgRcptHash:
			hash := v.Data.(*evmCommon.Hash)
			isnil, err := m.IsNil(hash, "rcpthash")
			if isnil {
				return err
			}
			rcpthash = *hash
		case actor.MsgGasUsed:
			gas := v.Data.(uint64)
			isnil, err := m.IsNil(gas, "gasused")
			if isnil {
				return err
			}
			gasused = gas
		case actor.MsgLocalParentInfo:
			parentinfo = v.Data.(*types.ParentInfo)
			isnil, err := m.IsNil(parentinfo, "parentinfo")
			if isnil {
				return err
			}
		case actor.MsgBlockParams:
			blockParams = v.Data.(*types.BlockParams)
			isnil, err := m.IsNil(blockParams, "blockParams")
			if isnil {
				return err
			}
		case actor.MsgBloom:
			bloom = v.Data.(evmTypes.Bloom)
			isnil, err := m.IsNil(bloom, "bloom")
			if isnil {
				return err
			}
		case actor.MsgWithDrawHash:
			withDrawHash = v.Data.(*evmCommon.Hash)

		}
	}

	m.CheckPoint("start makeBlock")
	m.AddLog(log.LogLevel_Info, "hashes", zap.String("Root", fmt.Sprintf("%x", accthash.Bytes())), zap.String("rcpthash", fmt.Sprintf("%x", rcpthash.Bytes())), zap.String("txhash", fmt.Sprintf("%x", txhash.Bytes())))

	// if len(txSelected) == 0 {
	// 	txhash = evmTypes.EmptyTxsHash
	// 	rcpthash = evmTypes.EmptyReceiptsHash
	// }

	header := CreateHerder(parentinfo, height, blockStart, accthash, gasused, txhash, rcpthash, blockParams, bloom, withDrawHash)
	block, err := CreateBlock(header, txSelected, m.SignerType)
	if err != nil {
		m.AddLog(log.LogLevel_Error, "block header eccode err", zap.String("err", err.Error()))
		return err
	}

	// save cache root and header hash
	currentinfo := &types.ParentInfo{
		ParentHash: header.Hash(),
		ParentRoot: accthash,
	}

	var na int
	intf.Router.Call("transactionalstore", "AddData", &transactional.AddDataRequest{
		Data:        currentinfo,
		RecoverFunc: "parentinfo",
	}, &na)

	m.MsgBroker.Send(actor.MsgAppHash, block.Hash())
	m.MsgBroker.Send(actor.MsgPendingBlock, block)
	m.MsgBroker.Send(actor.MsgParentInfo, currentinfo)
	m.MsgBroker.Send(actor.MsgLocalParentInfo, currentinfo)
	m.CheckPoint("send appHash")
	return nil
}

func CreateHerder(parentinfo *types.ParentInfo, height uint64, blockstart *actor.BlockStart, accthash evmCommon.Hash, gasused uint64, txhash evmCommon.Hash, rcpthash evmCommon.Hash, blockParams *types.BlockParams, bloom evmTypes.Bloom, withdrawhash *evmCommon.Hash) *evmTypes.Header {
	excessBlobGas := eip4844.CalcExcessBlobGas(0, 0)
	header := evmTypes.Header{
		ParentHash: parentinfo.ParentHash,
		Number:     big.NewInt(common.Uint64ToInt64(height)),
		GasLimit:   math.MaxUint32,

		Time:        blockstart.Timestamp.Uint64(),
		Difficulty:  evmCommon.Big0,
		Coinbase:    blockstart.Coinbase,
		Root:        accthash,
		GasUsed:     gasused,
		TxHash:      txhash,
		ReceiptHash: rcpthash,
		Extra:       blockstart.Extra,

		BaseFee:          big.NewInt(1),
		MixDigest:        blockParams.Random,
		BlobGasUsed:      new(uint64),
		ExcessBlobGas:    &excessBlobGas,
		ParentBeaconRoot: blockParams.BeaconRoot,

		UncleHash:       evmTypes.EmptyUncleHash,
		WithdrawalsHash: withdrawhash,
	}
	if gasused > 0 {
		header.Bloom = bloom
	}
	return &header
}

func CreateBlock(header *evmTypes.Header, txSelected [][]byte, SignerType uint8) (*types.MonacoBlock, error) {
	// ethHeader, err := evmRlp.EncodeToBytes(&header)
	ethHeader, err := header.MarshalJSON()
	if err != nil {
		return nil, err
	}

	headers := [][]byte{}
	ethHeaders := make([]byte, len(ethHeader)+1)
	bz := 0
	bz += copy(ethHeaders[bz:], []byte{types.AppType_Eth})
	bz += copy(ethHeaders[bz:], ethHeader)

	headers = append(headers, ethHeaders)

	block := &types.MonacoBlock{
		Blockhash: header.Hash().Bytes(),
		Height:    header.Number.Uint64(),
		Headers:   headers,
		Txs:       txSelected,
		Signer:    SignerType,
	}
	return block, nil
}
