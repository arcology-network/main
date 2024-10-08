/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package core

import (
	"fmt"
	"math"
	"math/big"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/storage/transactional"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
)

type MakeBlock struct {
	actor.WorkerThread
	ParentTime uint64
	// ParentHeader *evmTypes.Header
}

// return a Subscriber struct
func NewMakeBlock(concurrency int, groupid string) actor.IWorkerEx {
	in := MakeBlock{}
	in.Set(concurrency, groupid)
	return &in
}

func (m *MakeBlock) Inputs() ([]string, bool) {
	return []string{
		actor.MsgBlockStart,
		actor.MsgSelectedTxInfo,
		actor.MsgAcctHash,
		actor.MsgReceiptInfo,
		actor.MsgLocalParentInfo,
		actor.MsgBlockParams,
		actor.MsgWithDrawHash,
		actor.MsgSignerType,
		actor.MsgTransactionalAddCompleted,
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
	txhash := evmCommon.Hash{}
	accthash := evmCommon.Hash{}
	rcpthash := evmCommon.Hash{}
	gasused := uint64(0)
	txSelected := [][]byte{}
	parentinfo := &mtypes.ParentInfo{}
	height := uint64(0)
	// timestamp := big.NewInt(0)
	var blockParams *mtypes.BlockParams
	var blockStart *actor.BlockStart
	var bloom evmTypes.Bloom
	var withDrawHash *evmCommon.Hash
	var SignerType uint8
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgSignerType:
			SignerType = v.Data.(uint8)
		case actor.MsgBlockStart:
			blockStart = v.Data.(*actor.BlockStart)

			height = blockStart.Height
		case actor.MsgSelectedTxInfo:
			info := v.Data.(*mtypes.SelectedTxsInfo)
			isnil, err := m.IsNil(info, "txSelected")
			if isnil {
				return err
			}
			txhash = info.Txhash
			txSelected = info.Txs
		case actor.MsgAcctHash:
			// hash := v.Data.(*evmCommon.Hash)
			hash := v.Data.([32]byte)
			isnil, err := m.IsNil(hash, "accthash")
			if isnil {
				return err
			}
			accthash = evmCommon.BytesToHash([]byte(hash[:]))
			m.AddLog(log.LogLevel_Info, "received accthash", zap.String("accthash", fmt.Sprintf("%x", accthash)))
		case actor.MsgReceiptInfo:
			info := v.Data.(*mtypes.ReceiptInfo)
			isnil, err := m.IsNil(info, "receipt information")
			if isnil {
				return err
			}
			rcpthash = info.RcptHash
			gasused = info.Gasused
			bloom = info.BloomInfo
		case actor.MsgLocalParentInfo:
			parentinfo = v.Data.(*mtypes.ParentInfo)
			isnil, err := m.IsNil(parentinfo, "parentinfo")
			if isnil {
				return err
			}
		case actor.MsgBlockParams:
			blockParams = v.Data.(*mtypes.BlockParams)
			isnil, err := m.IsNil(blockParams, "blockParams")
			if isnil {
				return err
			}
		case actor.MsgWithDrawHash:
			withDrawHash = v.Data.(*evmCommon.Hash)
		case actor.MsgTransactionalAddCompleted:

		}
	}

	m.CheckPoint("start makeBlock")
	m.AddLog(log.LogLevel_Info, "hashes", zap.Uint64("gasused", gasused), zap.String("Root", fmt.Sprintf("%x", accthash.Bytes())), zap.String("rcpthash", fmt.Sprintf("%x", rcpthash.Bytes())), zap.String("txhash", fmt.Sprintf("%x", txhash.Bytes())))

	// if len(txSelected) == 0 {
	// 	txhash = evmTypes.EmptyTxsHash
	// 	rcpthash = evmTypes.EmptyReceiptsHash
	// }

	header := m.CreateHerder(parentinfo, height, blockStart, accthash, gasused, txhash, rcpthash, blockParams, bloom, withDrawHash)
	block, err := CreateBlock(header, txSelected, SignerType)
	if err != nil {
		m.AddLog(log.LogLevel_Error, "block header eccode err", zap.String("err", err.Error()))
		return err
	}

	// save cache root and header hash
	currentinfo := &mtypes.ParentInfo{
		ParentHash:    header.Hash(),
		ParentRoot:    accthash,
		ExcessBlobGas: *header.ExcessBlobGas,
		BlobGasUsed:   *header.BlobGasUsed,
	}
	// m.ParentHeader = header

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

	m.ParentTime = header.Time

	return nil
}

func (m *MakeBlock) CreateHerder(parentinfo *mtypes.ParentInfo, height uint64, blockstart *actor.BlockStart, accthash evmCommon.Hash, gasused uint64, txhash evmCommon.Hash, rcpthash evmCommon.Hash, blockParams *mtypes.BlockParams, bloom evmTypes.Bloom, withdrawhash *evmCommon.Hash) *evmTypes.Header {
	// var excessBlobGas uint64
	// if m.ParentHeader == nil {
	// 	excessBlobGas = eip4844.CalcExcessBlobGas(0, 0)
	// } else {
	excessBlobGas := eip4844.CalcExcessBlobGas(parentinfo.ExcessBlobGas, parentinfo.BlobGasUsed)
	// }

	headtime := blockstart.Timestamp.Uint64()
	if blockParams.Times > 0 {
		headtime = blockParams.Times
		if m.ParentTime >= headtime {
			headtime = m.ParentTime + 1
		}
	}

	header := evmTypes.Header{
		ParentHash: parentinfo.ParentHash,
		Number:     big.NewInt(common.Uint64ToInt64(height)),
		GasLimit:   math.MaxUint32,

		Time:        headtime,
		Difficulty:  evmCommon.Big0,
		Coinbase:    blockstart.Coinbase,
		Root:        accthash,
		GasUsed:     gasused,
		TxHash:      txhash,
		ReceiptHash: rcpthash,
		// Extra:       blockstart.Extra,

		BaseFee: big.NewInt(1),
		// MixDigest:        blockParams.Random,
		BlobGasUsed:      new(uint64),
		ExcessBlobGas:    &excessBlobGas,
		ParentBeaconRoot: blockParams.BeaconRoot,

		UncleHash:       evmTypes.EmptyUncleHash,
		WithdrawalsHash: withdrawhash,
	}
	if len(blockstart.Extra) != 0 && mtypes.RunAsL1 {
		header.Extra = blockstart.Extra
	}
	if blockParams.Random != (evmCommon.Hash{}) {
		header.MixDigest = blockParams.Random
	}
	if gasused > 0 {
		header.Bloom = bloom
	}
	return &header
}

func CreateBlock(header *evmTypes.Header, txSelected [][]byte, SignerType uint8) (*mtypes.MonacoBlock, error) {
	// ethHeader, err := evmRlp.EncodeToBytes(&header)
	ethHeader, err := header.MarshalJSON()
	if err != nil {
		return nil, err
	}

	headers := [][]byte{}
	ethHeaders := make([]byte, len(ethHeader)+1)
	bz := 0
	bz += copy(ethHeaders[bz:], []byte{mtypes.AppType_Eth})
	bz += copy(ethHeaders[bz:], ethHeader)

	headers = append(headers, ethHeaders)

	block := &mtypes.MonacoBlock{
		Blockhash: header.Hash().Bytes(),
		Height:    header.Number.Uint64(),
		Headers:   headers,
		Txs:       txSelected,
		Signer:    SignerType,
	}
	return block, nil
}
