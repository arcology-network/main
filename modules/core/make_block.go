package core

import (
	"fmt"
	"math/big"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethRlp "github.com/HPISTechnologies/3rd-party/eth/rlp"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/log"
	"github.com/HPISTechnologies/evm/params"
	"go.uber.org/zap"
)

type MakeBlock struct {
	actor.WorkerThread
}

//return a Subscriber struct
func NewMakeBlock(concurrency int, groupid string) actor.IWorkerEx {
	in := MakeBlock{}
	in.Set(concurrency, groupid)
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

	coinbase := ethCommon.Address{}
	txhash := ethCommon.Hash{}
	accthash := ethCommon.Hash{}
	rcpthash := ethCommon.Hash{}
	gasused := uint64(0)
	txSelected := [][]byte{}
	parentinfo := &types.ParentInfo{}
	height := uint64(0)
	timestamp := big.NewInt(0)
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgBlockStart:
			bs := v.Data.(*actor.BlockStart)
			timestamp = bs.Timestamp
			coinbase = bs.Coinbase
			height = bs.Height
		case actor.MsgSelectedTx:
			datas := v.Data.(types.Txs)
			isnil, err := m.IsNil(datas, "txSelected")
			if isnil {
				return err
			}
			txSelected = datas.Data // v.Data.([][]byte)
		case actor.MsgTxHash:
			hash := v.Data.(*ethCommon.Hash)
			isnil, err := m.IsNil(hash, "txhash")
			if isnil {
				return err
			}
			txhash = *hash
		case actor.MsgAcctHash:
			hash := v.Data.(*ethCommon.Hash)
			isnil, err := m.IsNil(hash, "accthash")
			if isnil {
				return err
			}
			accthash = *hash
			m.AddLog(log.LogLevel_Info, "received accthash", zap.String("accthash", fmt.Sprintf("%x", accthash)))
		case actor.MsgRcptHash:
			hash := v.Data.(*ethCommon.Hash)
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
		}
	}

	header := &ethTypes.Header{
		ParentHash: parentinfo.ParentHash,
		Number:     big.NewInt(common.Uint64ToInt64(height)),
		GasLimit:   uint64(params.GasLimit),
		//Extra:      w.extra,
		Time:        timestamp,
		Difficulty:  big.NewInt(1),
		Coinbase:    coinbase,
		Root:        accthash,
		GasUsed:     gasused,
		TxHash:      txhash,
		ReceiptHash: rcpthash,
	}

	m.AddLog(log.LogLevel_Info, "hashes", zap.String("Root", fmt.Sprintf("%x", accthash.Bytes())), zap.String("rcpthash", fmt.Sprintf("%x", rcpthash.Bytes())), zap.String("txhash", fmt.Sprintf("%x", txhash.Bytes())))
	//save cache root and header hash
	currentinfo := &types.ParentInfo{
		ParentHash: header.Hash(),
		ParentRoot: accthash,
	}

	//*******************************make block*******************************
	ethHeader, err := ethRlp.EncodeToBytes(&header)
	if err != nil {
		m.AddLog(log.LogLevel_Error, "block header eccode err", zap.String("err", err.Error()))
		return err
	}

	headers := [][]byte{}
	ethHeaders := make([]byte, len(ethHeader)+1)
	bz := 0
	bz += copy(ethHeaders[bz:], []byte{types.AppType_Eth})
	bz += copy(ethHeaders[bz:], ethHeader)

	headers = append(headers, ethHeaders)

	block := &types.MonacoBlock{
		Height:  height,
		Headers: headers,
		Txs:     txSelected,
	}

	m.MsgBroker.Send(actor.MsgAppHash, block.Hash())
	m.MsgBroker.Send(actor.MsgPendingBlock, block)
	m.MsgBroker.Send(actor.MsgParentInfo, currentinfo)
	m.MsgBroker.Send(actor.MsgLocalParentInfo, currentinfo)
	return nil
}
