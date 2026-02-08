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

package businessCase

import (
	"fmt"
	"math/big"
	"os"

	"github.com/arcology-network/common-lib/types"
	mconfig "github.com/arcology-network/main/config"
	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/actor/rpc"
	brokerpk "github.com/arcology-network/streamer/broker"
	jetlib "github.com/arcology-network/streamer/jet/lib"
	"github.com/arcology-network/streamer/logger"
	"github.com/ethereum/go-ethereum/common"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmcore "github.com/ethereum/go-ethereum/core"
	evmTypes "github.com/ethereum/go-ethereum/core/types"

	eushared "github.com/arcology-network/eu/shared"
	statestore "github.com/arcology-network/storage-committer"
)

func MakeStateStore(basePath string) (*evmcore.Genesis, *statestore.StateStore, []*univaluepk.Univalue) {
	gen := storage.ReadGenesis("./genesis.json")
	store, _, uinvalues := storage.InitGenesisAccounts(basePath+"/db", gen, 0)

	return gen, store, uinvalues
}

func MakeTxAccessRecordSet(txhashes []evmCommon.Hash, Ids []uint64, univalues []*univaluepk.Univalue) *eushared.TxAccessRecordSet {
	aces := make([]*eushared.TxAccessRecords, len(txhashes))
	for i := range txhashes {
		aces[i] = &eushared.TxAccessRecords{
			Hash:     string(txhashes[i][:]),
			ID:       Ids[i],
			Accesses: univalues[i*10 : (i+1)*10],
		}
	}
	tarss := eushared.TxAccessRecordSet(aces)
	return &tarss
}

func Transfer(block *mtypes.MonacoBlock, txhashes []evmCommon.Hash) []*types.StandardMessage {
	sys_Signer := uint8(0)
	ChainId := big.NewInt(118)

	stdmsgs := make([]*types.StandardMessage, len(block.Txs))
	for idx := range block.Txs {
		otx := new(evmTypes.Transaction)
		if err := otx.UnmarshalBinary(block.Txs[idx][1:]); err != nil {
			continue
		}

		stdTx := &types.StandardTransaction{
			TxHash:            txhashes[idx],
			NativeTransaction: otx,
			Signer:            sys_Signer,
		}
		signer := mtypes.MakeSigner(sys_Signer, ChainId)
		err := stdTx.UnSign(signer)
		if err != nil {
			continue
		}

		stdmsgs[idx] = &types.StandardMessage{
			ID:     uint64(idx + 1),
			TxHash: txhashes[idx],
			Native: stdTx.NativeMessage,
			Source: 0,
		}

	}
	return stdmsgs
}
func MakeMonacoBlock() (*mtypes.MonacoBlock, []evmCommon.Hash) {
	txs := [][]byte{
		evmCommon.Hex2Bytes("0002f8667601010382c24294b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f8084a523b88ac080a025665926933f352bcac4d9665cf78b930a9fa6089939b64385ab463b9a8b84d8a01999d62d473e0baf86c48c92f9b1f9462ea0882d870dff879089ca177fc681bc"),
		evmCommon.Hex2Bytes("0002f8667601010382c24294b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f8084a523b88ac080a0b72738d59e994e96aa51d20440b856aaf491c9020c8f884b1692c309d02a76bea03d93532ddab895d464c9640988e5dec07bd770e31a6cb4c6b5d2e2f4125040c6"),
	}
	blockHash := evmCommon.BytesToHash([]byte{101, 102, 103, 104, 105, 106, 107, 108})
	txhashes := []evmCommon.Hash{
		evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		evmCommon.BytesToHash([]byte{11, 12, 13, 14, 51, 16, 17, 18}),
	}
	return &mtypes.MonacoBlock{
		Height:    10,
		Blockhash: blockHash.Bytes(),
		Headers:   [][]byte{},
		Txs:       txs,
		Signer:    0,
	}, txhashes
}

func MakeReceipts() []*evmTypes.Receipt {
	blockHeight := big.NewInt(10)
	txhashes := []evmCommon.Hash{
		evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		evmCommon.BytesToHash([]byte{11, 12, 13, 14, 51, 16, 17, 18}),
	}
	blockHash := evmCommon.BytesToHash([]byte{101, 102, 103, 104, 105, 106, 107, 108})
	return []*evmTypes.Receipt{
		{
			Type:              evmTypes.LegacyTxType,
			PostState:         []byte{1},
			Status:            evmTypes.ReceiptStatusSuccessful,
			CumulativeGasUsed: 78565466,
			Bloom:             evmTypes.BytesToBloom([]byte{1, 23, 4, 5, 6, 78}),
			Logs: []*evmTypes.Log{
				{
					Address: common.BytesToAddress([]byte{0x33}),
					// derived fields:
					BlockNumber: blockHeight.Uint64(),
					TxHash:      txhashes[0],
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       0,
				},
				{
					Address: common.BytesToAddress([]byte{0x03, 0x33}),
					// derived fields:
					BlockNumber: blockHeight.Uint64(),
					TxHash:      txhashes[0],
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       1,
				},
			},
			TxHash:           txhashes[0],
			BlockHash:        blockHash,
			BlockNumber:      blockHeight,
			TransactionIndex: 0,
		},
		{
			Type:              evmTypes.LegacyTxType,
			PostState:         []byte{1},
			Status:            evmTypes.ReceiptStatusSuccessful,
			CumulativeGasUsed: 64656,
			Bloom:             evmTypes.BytesToBloom([]byte{11, 231, 4, 15, 61, 78}),
			Logs: []*evmTypes.Log{
				{
					Address: common.BytesToAddress([]byte{0x33}),
					// derived fields:
					BlockNumber: blockHeight.Uint64(),
					TxHash:      txhashes[1],
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       0,
				},
				{
					Address: common.BytesToAddress([]byte{0x03, 0x33}),
					// derived fields:
					BlockNumber: blockHeight.Uint64(),
					TxHash:      txhashes[1],
					TxIndex:     0,
					BlockHash:   blockHash,
					Index:       1,
				},
			},
			TxHash:           txhashes[1],
			BlockHash:        blockHash,
			BlockNumber:      blockHeight,
			TransactionIndex: 1,
		},
	}
}

func ClearPath(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Printf("remove failed: %v\n", err)
		return
	}
}

func InitCfg(basepath, globalConfigFile, jetConfigFile, appConfigFile string) (*mconfig.AppConfig, *brokerpk.StatefulStreamer) {
	ss := jetlib.RunJetTestServer()

	globalConfig, _ := mconfig.LoadGlobalConfig(globalConfigFile)
	jetConfig, _ := jetlib.LoadConfig(jetConfigFile)
	appConfig, _ := mconfig.LoadAppConfig(appConfigFile)

	logger.InitLog("./log.toml", basepath+"/log/app.log")
	jetConfig.Nats.Servers[0] = ss.ClientURL()

	broker := brokerpk.NewStatefulStreamer()
	rpc.InitGlobalRPCFactory()
	rpc.InitGlobalRPCClient(broker, globalConfig.RpcConcurrent, globalConfig.RpcTimeoutSeconds)

	appConfig.InitApp(broker, globalConfig, jetConfig)

	return appConfig, broker
}

func StartSys(appcfg *mconfig.AppConfig, broker *brokerpk.StatefulStreamer) {
	broker.Serve()

	for _, worker := range appcfg.WorkersDict {
		if _, ok := worker.(actor.Initializer); ok {
			msgs := worker.(actor.Initializer).InitMsgs()
			for _, msg := range msgs {
				broker.Send(msg.Name, msg)
			}
		}
	}

	for _, msg := range appcfg.StartMsgs {
		broker.Send(msg.Name, &msg)
	}
}
