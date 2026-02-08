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
	types "github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type MakeBlock struct {
	ParentTime  uint64
	currentinfo *mtypes.ParentInfo
	block       *mtypes.MonacoBlock
	header      *evmTypes.Header
}

// return a Subscriber struct
func NewMakeBlock() actor.Business {
	in := MakeBlock{}
	return &in
}

func (m *MakeBlock) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgBlockStart,
		scommon.MsgSelectedTxInfo,
		scommon.MsgAcctHash,
		scommon.MsgReceiptInfo,
		scommon.MsgLocalParentInfo,
		scommon.MsgBlockParams,
		scommon.MsgWithDrawHash,
		scommon.MsgSignerType,
		scommon.MsgGenerationReapingCompleted,
		scommon.MsgInclusive,
	}, true
}

func (m *MakeBlock) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgAppHash:         1,
		scommon.MsgParentInfo:      1,
		scommon.MsgLocalParentInfo: 1,
		scommon.MsgPendingBlock:    1,
	}
}

func (m *MakeBlock) PrimaryMsg() string {
	return scommon.MsgInclusive
}

func (m *MakeBlock) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgBlockStart, m.makeBlock)
	reg.Register(scommon.MsgSelectedTxInfo, m.makeBlock)
	reg.Register(scommon.MsgAcctHash, m.makeBlock)
	reg.Register(scommon.MsgReceiptInfo, m.makeBlock)
	reg.Register(scommon.MsgLocalParentInfo, m.makeBlock)
	reg.Register(scommon.MsgBlockParams, m.makeBlock)
	reg.Register(scommon.MsgWithDrawHash, m.makeBlock)
	reg.Register(scommon.MsgSignerType, m.makeBlock)
	reg.Register(scommon.MsgGenerationReapingCompleted, m.makeBlock)
	reg.Register(scommon.MsgInclusive, m.makeBlock)
	reg.Register("sendResult", m.sendResult)
}

func (m *MakeBlock) makeBlock(ctx *actor.ActionContext) error {
	txhash := evmCommon.Hash{}
	accthash := evmCommon.Hash{}
	rcpthash := evmCommon.Hash{}
	gasused := uint64(0)
	txSelected := [][]byte{}
	parentinfo := &mtypes.ParentInfo{}
	height := uint64(0)
	inclusivelist := []evmCommon.Hash{}
	var selectedInfo *mtypes.SelectedTxsInfo
	// timestamp := big.NewInt(0)
	var blockParams *mtypes.BlockParams
	var blockStart *actor.BlockStart
	var bloom evmTypes.Bloom
	var withDrawHash *evmCommon.Hash
	var SignerType uint8

	for _, v := range ctx.Messages {
		switch v.Name {
		case scommon.MsgSignerType:
			SignerType = v.Data.(uint8)
		case scommon.MsgBlockStart:
			blockStart = v.Data.(*actor.BlockStart)
			height = blockStart.Height
		case scommon.MsgSelectedTxInfo:
			selectedInfo = v.Data.(*mtypes.SelectedTxsInfo)
			txhash = selectedInfo.Txhash
		case scommon.MsgAcctHash:
			hash := v.Data.([32]byte)
			accthash = evmCommon.BytesToHash([]byte(hash[:]))
			ctx.ExecCtx.LogInfo("received accthash", logger.F("accthash", fmt.Sprintf("%x", accthash)))
		case scommon.MsgReceiptInfo:
			info := v.Data.(*mtypes.ReceiptInfo)
			rcpthash = info.RcptHash
			gasused = info.Gasused
			bloom = info.BloomInfo
		case scommon.MsgLocalParentInfo:
			parentinfo = v.Data.(*mtypes.ParentInfo)
		case scommon.MsgBlockParams:
			blockParams = v.Data.(*mtypes.BlockParams)
		case scommon.MsgWithDrawHash:
			withDrawHash = v.Data.(*evmCommon.Hash)
		case scommon.MsgGenerationReapingCompleted:
		case scommon.MsgInclusive:
			inclusivelist = v.Data.(*types.InclusiveList).HashList
		}
	}

	ctx.ExecCtx.LogInfo("start makeBlock", logger.F("gasused", gasused), logger.F("Root", fmt.Sprintf("%x", accthash.Bytes())), logger.F("txhash", fmt.Sprintf("%x", txhash.Bytes())))

	txSelected = OrderTxs(selectedInfo, inclusivelist)

	header := m.CreateHerder(parentinfo, height, blockStart, accthash, gasused, txhash, rcpthash, blockParams, bloom, withDrawHash)
	var err error
	m.block, err = CreateBlock(header, txSelected, SignerType)
	if err != nil {
		ctx.ExecCtx.LogErr("block header eccode err", logger.F("err", err.Error()))
		return err
	}

	// save cache root and header hash
	m.currentinfo = &mtypes.ParentInfo{
		ParentHash:    header.Hash(),
		ParentRoot:    accthash,
		ExcessBlobGas: *header.ExcessBlobGas,
		BlobGasUsed:   *header.BlobGasUsed,
	}

	m.header = header
	ctx.ExecCtx.InvokeRPC("transactionalstore", "AddData", &transactional.AddDataRequest{
		Data:        m.currentinfo,
		RecoverFunc: "parentinfo",
	}, "sendResult")

	return nil
}

func (m *MakeBlock) sendResult(ctx *actor.ActionContext) error {
	ctx.ExecCtx.Send(scommon.MsgAppHash, m.block.Hash())
	ctx.ExecCtx.Send(scommon.MsgPendingBlock, m.block)
	ctx.ExecCtx.Send(scommon.MsgParentInfo, m.currentinfo)
	ctx.ExecCtx.Send(scommon.MsgLocalParentInfo, m.currentinfo)

	ctx.ExecCtx.LogInfo("send appHash")
	m.ParentTime = m.header.Time
	return nil
}

func OrderTxs(selectedInfo *mtypes.SelectedTxsInfo, inclusivelist []evmCommon.Hash) [][]byte {
	mmp := make(map[evmCommon.Hash]int, len(inclusivelist))
	for i := range selectedInfo.HashList {
		mmp[selectedInfo.HashList[i]] = i
	}
	txSelected := make([][]byte, len(inclusivelist))
	for i := range inclusivelist {
		txSelected[i] = selectedInfo.Txs[mmp[inclusivelist[i]]]
	}
	return txSelected
}

func (m *MakeBlock) CreateHerder(parentinfo *mtypes.ParentInfo, height uint64, blockstart *actor.BlockStart, accthash evmCommon.Hash, gasused uint64, txhash evmCommon.Hash, rcpthash evmCommon.Hash, blockParams *mtypes.BlockParams, bloom evmTypes.Bloom, withdrawhash *evmCommon.Hash) *evmTypes.Header {
	excessBlobGas := eip4844.CalcExcessBlobGas(parentinfo.ExcessBlobGas, parentinfo.BlobGasUsed)

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
