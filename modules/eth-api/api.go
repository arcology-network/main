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

package ethapi

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
	internal "github.com/arcology-network/main/modules/eth-api/backend"
	wal "github.com/arcology-network/main/modules/eth-api/wallet"
	mtypes "github.com/arcology-network/main/types"
	jsonrpc "github.com/deliveroo/jsonrpc-go"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/rs/cors"
)

func LoadCfg(tomfile string, conf interface{}) {
	_, err := toml.DecodeFile(tomfile, conf)
	if err != nil {
		panic("load conf err :" + err.Error())
	}
}

type Options struct {
	KeyFile         string `short:"k" long:"keyfile" description:"Private keys file path" default:"./accounts.txt"`
	ChainID         uint64 `short:"c" long:"chainid" description:"Network chain ID"`
	Port            uint64 `short:"p" long:"port" description:"Service port" default:"7545"`
	Debug           bool   `short:"d" long:"debug" description:"Enable debug mode"`
	Waits           int    `short:"w" long:"waits" description:"wait seconds when query receipt" default:"60"`
	Coinbase        string `short:"cb" long:"coinbase" description:"coinbase address of node`
	ProtocolVersion int    `short:"pv" long:"protocolVersion" description:"Protocol Version`
	Hashrate        int    `short:"hr" long:hashrate" description:"hash rate`
	ExecId          string `short:"EID" long:ExecID" description:"Exector instance id `
	AuthPort        uint64 `short:"ap" long:"authport" description:"auth Service port" default:"8551"`
	JwtFile         string `short:"j" long:"jwtfile" description:"Jwt file path" default:"./jwt.txt"`
}

var options Options
var backend internal.EthereumAPI
var wallet *wal.Wallet

func version(ctx context.Context) (interface{}, error) {
	//return options.ChainID, nil
	//return NumberToHex(options.ChainID), nil
	return fmt.Sprintf("%d", options.ChainID), nil
}

func chainId(ctx context.Context) (interface{}, error) {
	return NumberToHex(options.ChainID), nil
}

func blockNumber(ctx context.Context) (interface{}, error) {
	number, err := backend.BlockNumber()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}

	return NumberToHex(number), nil
}

func getHeaderByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}

	block, err := backend.GetHeaderByNumber(number)
	if err != nil || block.Header == nil {
		return nil, nil
	}
	return RPCMarshalHeader(block.Header), nil
}

func getBlockByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}

	isTransaction, err := ToBool(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid isTransaction  given :%v", err)
	}

	block, err := backend.GetBlockByNumber(number, isTransaction)
	if err != nil {
		return nil, nil
	}
	return parseBlock(block, isTransaction), nil
}

func getHeaderByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}

	block, err := backend.GetHeaderByHash(hash)
	if err != nil || block.Header == nil {
		return nil, nil
	}
	return RPCMarshalHeader(block.Header), nil
}

func getBlockByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}

	isTransaction, err := ToBool(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid isTransaction  given :%v", err)
	}

	block, err := backend.GetBlockByHash(hash, isTransaction)
	if err != nil {
		return nil, nil
	}
	return parseBlock(block, isTransaction), nil
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *ethtyp.Header) map[string]interface{} {
	result := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             head.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"extraData":        hexutil.Bytes(head.Extra),
		"gasLimit":         hexutil.Uint64(head.GasLimit),
		"gasUsed":          hexutil.Uint64(head.GasUsed),
		"timestamp":        hexutil.Uint64(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
	}
	if head.BaseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(head.BaseFee)
	}
	if head.WithdrawalsHash != nil {
		result["withdrawalsRoot"] = head.WithdrawalsHash
	}
	if head.BlobGasUsed != nil {
		result["blobGasUsed"] = hexutil.Uint64(*head.BlobGasUsed)
	}
	if head.ExcessBlobGas != nil {
		result["excessBlobGas"] = hexutil.Uint64(*head.ExcessBlobGas)
	}
	if head.ParentBeaconRoot != nil {
		result["parentBeaconBlockRoot"] = head.ParentBeaconRoot
	}
	return result
}
func parseBlock(block *mtypes.RPCBlock, isTransaction bool) interface{} {
	if block.Header == nil {
		return nil
	}
	uncles := make([]string, 0)
	header := block.Header

	blockResult := RPCMarshalHeader(header)
	blockResult["uncles"] = uncles

	// blockResult := map[string]interface{}{
	// 	"uncles": uncles,

	// 	"number":           (*hexutil.Big)(header.Number),
	// 	"hash":             header.Hash(),
	// 	"parentHash":       header.ParentHash,
	// 	"nonce":            header.Nonce,
	// 	"mixHash":          header.MixDigest,
	// 	"sha3Uncles":       header.UncleHash,
	// 	"logsBloom":        header.Bloom,
	// 	"stateRoot":        header.Root,
	// 	"miner":            header.Coinbase,
	// 	"difficulty":       (*hexutil.Big)(header.Difficulty),
	// 	"extraData":        hexutil.Bytes(header.Extra),
	// 	"size":             hexutil.Uint64(header.Size()),
	// 	"gasLimit":         hexutil.Uint64(header.GasLimit),
	// 	"gasUsed":          hexutil.Uint64(header.GasUsed),
	// 	"timestamp":        hexutil.Uint64(header.Time),
	// 	"transactionsRoot": header.TxHash,
	// 	"receiptsRoot":     header.ReceiptHash,
	// 	"totalDifficulty":  (*hexutil.Big)(header.Difficulty),
	// 	// "transactions":     transactions,
	// }

	if isTransaction {
		transactions := make([]*mtypes.RPCTransaction, len(block.Transactions))
		for i := range block.Transactions {
			transactions[i] = AttachChainId(block.Transactions[i].(*mtypes.RPCTransaction), options.ChainID)
		}
		blockResult["transactions"] = transactions
	} else {
		hashes := make([]string, len(block.Transactions))
		for i := range block.Transactions {
			hashes[i] = fmt.Sprintf("%x", block.Transactions[i])
		}
		blockResult["transactions"] = hashes
	}

	return blockResult
}

func getTransactionCount(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given :%v", err)
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}

	nonce, err := backend.GetTransactionCount(address, number)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(nonce), nil
}

func getCode(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given :%v", err)
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}

	code, err := backend.GetCode(address, number)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(code), nil
}

func getBalance(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given :%v", err)
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}

	balance, err := backend.GetBalance(address, number)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(balance), nil
}

func getStorageAt(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given :%v", err)
	}

	key, err := ToHash(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}

	number, err := ToBlockNumber(params[2])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}

	value, err := backend.GetStorageAt(address, key.Hex(), number)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(value), nil
}

func accounts(ctx context.Context) (interface{}, error) {
	return wallet.Accounts(), nil
}

func estimateGas(ctx context.Context, params []interface{}) (interface{}, error) {
	msg, err := ToCallMsg(params[0], true)
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid call msg given :%v", err)
	}

	gas, _ := backend.EstimateGas(msg)
	return NumberToHex(gas), nil
}

func gasPrice(ctx context.Context) (interface{}, error) {
	gp, err := backend.GasPrice()
	if err != nil {
		return nil, nil
	}
	return NumberToHex(gp), nil
}

func sendTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	tx, err := ToSendTxArgs(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid transaction given :%v", err)
	}

	rawTx, err := wallet.SignTx(0, tx.Nonce, tx.To, tx.Value, tx.Gas, tx.GasPrice, tx.Data)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}

	hash, err := backend.SendRawTransaction(rawTx)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return hash.Hex(), nil
}

func getBlockReceipts(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}
	var receipts []*ethtyp.Receipt
	receipts, err = backend.GetBlockReceipts(uint64(number))
	if err != nil {
		return nil, nil
	}
	allfields := make([]map[string]interface{}, len(receipts))
	for i := range receipts {
		tx, err := backend.GetTransactionByHash(receipts[i].TxHash)
		if err != nil {
			return nil, nil
		}
		allfields[i] = marshalReceipt(receipts[i], tx)
	}
	return allfields, nil
}

func getTransactionReceipt(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}
	var receipt *ethtyp.Receipt

	// receipt, err = backend.GetTransactionReceipt(hash)
	// if receipt == nil {
	// 	return "null", nil
	// }
	queryCounter := options.Waits
	for queryIdx := 0; queryIdx < queryCounter; queryIdx++ {
		receipt, err = backend.GetTransactionReceipt(hash)
		if err != nil {
			if queryIdx == queryCounter-1 {
				return nil, nil
			}
			time.Sleep(time.Duration(time.Second * 1))
		} else {
			break
		}
	}

	tx, err := backend.GetTransactionByHash(hash)
	if err != nil {
		return nil, nil
	}

	return marshalReceipt(receipt, tx), nil
}
func marshalReceipt(receipt *ethtyp.Receipt, tx *mtypes.RPCTransaction) map[string]interface{} {
	fields := map[string]interface{}{
		"blockHash":         receipt.BlockHash,
		"blockNumber":       hexutil.Uint64(receipt.BlockNumber.Uint64()),
		"transactionHash":   receipt.TxHash,
		"transactionIndex":  hexutil.Uint64(receipt.TransactionIndex),
		"from":              tx.From,
		"to":                tx.To,
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
		"type":              hexutil.Uint(receipt.Type),
	}
	//fields["effectiveGasPrice"] = hexutil.Uint64(receipt.)
	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = []*ethtyp.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}

	fields["effectiveGasPrice"] = hexutil.Uint64(tx.GasPrice.ToInt().Uint64())
	return fields
}

func getTransactionByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}

	tx, err := backend.GetTransactionByHash(hash)
	if err != nil {
		return nil, nil
	}

	return ToTransactionResponse(tx, options.ChainID), nil
}

func sendRawTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	bytes, err := ToBytes(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid raw transaction given :%v", err)
	}

	hash, err := backend.SendRawTransaction(bytes)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return hash.Hex(), nil
}
func sendRawTransactions(ctx context.Context, params []interface{}) (interface{}, error) {
	txs := make([][]byte, len(params))
	for i := range params {
		bytes, err := ToBytes(params[i])
		if err != nil {
			return nil, jsonrpc.InvalidParams("invalid raw transaction given :%v", err)
		}
		txs[i] = bytes
	}
	result, err := backend.SendRawTransactions(txs)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return result, nil
}

func call(ctx context.Context, params []interface{}) (interface{}, error) {
	msg, err := ToCallMsg(params[0], false)
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid call msg given : %v", err)
	}

	ret, err := backend.Call(msg)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(ret), nil
}

func getLogs(ctx context.Context, params []interface{}) (interface{}, error) {
	filter, err := ToFilter(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given : %v", err)
	}

	logs, err := backend.GetLogs(filter)
	if err != nil {
		return nil, nil
	}
	return logs, nil
}

func getBlockTransactionCountByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}

	txsNum, err := backend.GetBlockTransactionCountByHash(hash)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(txsNum), nil
}

func getBlockTransactionCountByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid number given : %v", err)
	}

	txsNum, err := backend.GetBlockTransactionCountByNumber(number)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(txsNum), nil
}

func getTransactionByBlockHashAndIndex(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}

	index, err := ToBlockIndex(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid index given :%v", err)
	}

	tx, err := backend.GetTransactionByBlockHashAndIndex(hash, index)
	if err != nil || tx == nil {
		return nil, nil
	}
	return ToTransactionResponse(tx, options.ChainID), nil
}
func getTransactionByBlockNumberAndIndex(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}
	index, err := ToBlockIndex(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block index given :%v", err)
	}

	tx, err := backend.GetTransactionByBlockNumberAndIndex(number, index)
	if err != nil || tx == nil {
		return nil, nil
	}
	return ToTransactionResponse(tx, options.ChainID), nil
}
func getUncleCountByBlockHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given :%v", err)
	}

	txsNum, err := backend.GetUncleCountByBlockHash(hash)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(txsNum), nil
}
func getUncleCountByBlockNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given :%v", err)
	}

	txsNum, err := backend.GetUncleCountByBlockNumber(number)
	if err != nil {
		return nil, nil
	}
	return NumberToHex(txsNum), nil
}

func submitWork(ctx context.Context) (interface{}, error) {

	// ok, err := backend.SubmitWork()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return fmt.Sprintf("%v", true), nil
}
func submitHashrate(ctx context.Context) (interface{}, error) {

	// ok, err := backend.SubmitHashrate()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return fmt.Sprintf("%v", true), nil
}
func hashrate(ctx context.Context) (interface{}, error) {

	// hashrate, err := backend.Hashrate()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return NumberToHex(options.Hashrate), nil
}
func getWork(ctx context.Context) (interface{}, error) {

	// works, err := backend.GetWork()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return []string{}, nil
}
func protocolVersion(ctx context.Context) (interface{}, error) {

	// version, err := backend.ProtocolVersion()
	// if err != nil {
	// 	return nil, jsonrpc.InternalError(err)
	// }
	return fmt.Sprintf("%v", options.ProtocolVersion), nil
}
func coinbase(ctx context.Context) (interface{}, error) {
	return options.Coinbase, nil
}

func sign(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given :%v", err)
	}
	txdata, err := ToBytes(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid bytes given :%v", err)
	}
	retData, err := wallet.Sign(address, txdata)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(retData), nil
}

func signTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	tx, err := ToSendTxArgs(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid transaction given :%v", err)
	}
	rawTx, err := wallet.SignTx(0, tx.Nonce, tx.To, tx.Value, tx.Gas, tx.GasPrice, tx.Data)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(rawTx), nil
}
func feeHistory(ctx context.Context) (interface{}, error) {
	nilbig := (*hexutil.Big)(big.NewInt(0))
	results := mtypes.FeeHistoryResult{
		OldestBlock:  nilbig,
		GasUsedRatio: []float64{0.0},
		Reward: [][]*hexutil.Big{
			{nilbig},
		},
		BaseFee: []*hexutil.Big{nilbig},
	}
	return results, nil
}
func syncing(ctx context.Context) (interface{}, error) {
	ok, err := backend.Syncing()
	if err != nil {
		return nil, nil
	}
	return fmt.Sprintf("%v", ok), nil
}
func mining(ctx context.Context) (interface{}, error) {
	ok, err := backend.Proposer()
	if err != nil {
		return nil, nil
	}
	return fmt.Sprintf("%v", ok), nil
}

func newFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	filter, err := ToFilter(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given :%v", err)
	}

	id, err := backend.NewFilter(filter)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return id, nil
}

func newBlockFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := backend.NewBlockFilter()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return id, nil
}
func newPendingTransactionFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := backend.NewPendingTransactionFilter()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return id, nil
}
func uninstallFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ToID(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given :%v", err)
	}
	ok, err := backend.UninstallFilter(id)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return ok, nil
}

func getFilterChanges(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ToID(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given :%v", err)
	}
	results, err := backend.GetFilterChanges(id)
	if err != nil {
		return nil, nil
	}
	return results, nil
}
func getFilterLogs(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ToID(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given :%v", err)
	}

	logs, err := backend.GetFilterLogs(id)
	if err != nil {
		return nil, nil
	}
	return logs, nil
}

func clientVersion(ctx context.Context) (interface{}, error) {
	return fmt.Sprintf("client-%d", options.ChainID), nil
}

func txpoolContent(ctx context.Context) (interface{}, error) {
	mm := map[string]interface{}{}
	blockResult := map[string]interface{}{
		"pending": mm,
		"queued":  mm,
	}
	return blockResult, nil
}
func traceTransaction(ctx context.Context) (interface{}, error) {
	return nil, nil
}
func maxPriorityFeePerGas(ctx context.Context) (interface{}, error) {
	return big.NewInt(1), nil
}
func forkchoiceUpdatedV2(ctx context.Context, params []interface{}) (interface{}, error) {
	update, err := ParseJsonParam[engine.ForkchoiceStateV1](params[0], "ForkchoiceStateV1")
	if err != nil {
		return nil, err
	}
	var payloadAttributes *engine.PayloadAttributes
	if len(params) > 1 && params[1] != nil {
		payloadAttributes, err = ParseJsonParam[engine.PayloadAttributes](params[1], "payloadAttributes")
		if err != nil {
			return nil, err
		}
	}
	return backend.ForkchoiceUpdatedV2(*update, payloadAttributes, options.ChainID)
}
func forkchoiceUpdatedV3(ctx context.Context, params []interface{}) (interface{}, error) {
	update, err := ParseJsonParam[engine.ForkchoiceStateV1](params[0], "ForkchoiceStateV1")
	if err != nil {
		return nil, err
	}
	var payloadAttributes *engine.PayloadAttributes
	if len(params) > 1 && params[1] != nil {
		payloadAttributes, err = ParseJsonParam[engine.PayloadAttributes](params[1], "payloadAttributes")
		if err != nil {
			return nil, err
		}
	}

	if payloadAttributes != nil {
		// TODO(matt): according to https://github.com/ethereum/execution-apis/pull/498,
		// payload attributes that are invalid should return error
		// engine.InvalidPayloadAttributes. Once hive updates this, we should update
		// on our end.
		if payloadAttributes.Withdrawals == nil {
			return engine.STATUS_INVALID, engine.InvalidParams.With(errors.New("missing withdrawals"))
		}
		if payloadAttributes.BeaconRoot == nil {
			return engine.STATUS_INVALID, engine.InvalidParams.With(errors.New("missing beacon root"))
		}
		// if api.eth.BlockChain().Config().LatestFork(payloadAttributes.Timestamp) != forks.Cancun {
		// 	return engine.STATUS_INVALID, engine.UnsupportedFork.With(errors.New("forkchoiceUpdatedV3 must only be called for cancun payloads"))
		// }
	}

	return backend.ForkchoiceUpdatedV2(*update, payloadAttributes, options.ChainID)
}

func getPayloadV2(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ParseJsonParam[engine.PayloadID](params[0], "PayloadID")
	if err != nil {
		return nil, err
	}
	return backend.GetPayloadV2(*id)
}

func getPayloadV3(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ParseJsonParam[engine.PayloadID](params[0], "PayloadID")
	if err != nil {
		return nil, err
	}
	return backend.GetPayloadV2(*id)
}

func newPayloadV3(ctx context.Context, params []interface{}) (interface{}, error) {
	payload, err := ParseJsonParam[engine.ExecutableData](params[0], "ExecutableData")
	if err != nil {
		return nil, err
	}

	if payload.Withdrawals == nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil withdrawals post-shanghai"))
	}
	if payload.ExcessBlobGas == nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil excessBlobGas post-cancun"))
	}
	if payload.BlobGasUsed == nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil blobGasUsed post-cancun"))
	}

	// if versionedHashes == nil {
	// 	return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil versionedHashes post-cancun"))
	// }
	// if beaconRoot == nil {
	// 	return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil beaconRoot post-cancun"))
	// }

	// if api.eth.BlockChain().Config().LatestFork(params.Timestamp) != forks.Cancun {
	// 	return engine.PayloadStatusV1{Status: engine.INVALID}, engine.UnsupportedFork.With(errors.New("newPayloadV3 must only be called for cancun payloads"))
	// }

	return backend.NewPayloadV2(*payload)
}

func newPayloadV2(ctx context.Context, params []interface{}) (interface{}, error) {
	payload, err := ParseJsonParam[engine.ExecutableData](params[0], "ExecutableData")
	if err != nil {
		return nil, err
	}
	return backend.NewPayloadV2(*payload)
}
func exchangeTransitionConfigurationV1(ctx context.Context, params []interface{}) (interface{}, error) {
	signal, err := ParseJsonParam[catalyst.SuperchainSignal](params[0], "SuperchainSignal")
	if err != nil {
		return nil, err
	}
	return backend.SignalSuperchainV1(signal)
}

func getProof(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given :%v", err)
	}
	keys, err := ToKeys(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid keys given :%v", err)
	}
	blockParameter, err := mtypes.ParseBlockParameter(params[2])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid blockParameter given :%v", err)
	}
	// blockTag, err := ToHash(params[2])
	// if err != nil {
	// 	return nil, jsonrpc.InvalidParams("invalid blockTag given %v", params[2])
	// }
	request := &mtypes.RequestProof{
		Address:        address,
		Keys:           keys,
		BlockParameter: blockParameter,
	}
	return backend.GetProof(request)
}
func ToKeys(param interface{}) ([]common.Hash, error) {
	if params, ok := param.([]interface{}); !ok {
		return []common.Hash{}, errors.New("unexpected data type given,address")
	} else {
		keys := make([]common.Hash, len(params))
		for i := range keys {
			key, err := ToHash(params[i])
			if err != nil {
				return []common.Hash{}, errors.New("unexpected data type given,hash")
			}
			keys[i] = key
		}
		return keys, nil
	}
}

func startJsonRpc() {
	filters := internal.NewFilters()

	privateKeys := LoadKeys(options.KeyFile)

	// logFile, err := os.OpenFile("./rpc.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	// if err != nil {
	// 	fmt.Println("open log file failed, err:", err)
	// 	return
	// }
	// log.SetOutput(logFile)
	// log.SetFlags(log.Lmicroseconds | log.Ldate)

	server := jsonrpc.New()
	server.Use(func(next jsonrpc.Next) jsonrpc.Next {
		return func(ctx context.Context, params interface{}) (interface{}, error) {
			// method := jsonrpc.MethodFromContext(ctx)
			// fmt.Printf("***********************************************method: %v \t params:%v \n", method, params)
			return next(ctx, params)
		}
	})

	server.Register(methods)

	if options.Debug {
		backend = internal.NewEthereumAPIMock(new(big.Int).SetUint64(options.ChainID))
	} else {
		backend = internal.NewMonaco(filters)
	}

	wallet = wal.NewWallet(new(big.Int).SetUint64(options.ChainID), privateKeys)

	c := cors.AllowAll()
	go http.ListenAndServe(fmt.Sprintf(":%d", options.Port), c.Handler(server))
}

var methods = map[string]jsonrpc.MethodFunc{
	"net_version":               version,
	"eth_chainId":               chainId, //mock_chainId, //
	"eth_blockNumber":           blockNumber,
	"eth_getBlockByNumber":      getBlockByNumber, //mock_getBlockByNumber, //
	"eth_getBlockByHash":        getBlockByHash,
	"eth_getTransactionCount":   getTransactionCount, //mock_getTransactionCount, //
	"eth_getCode":               getCode,
	"eth_getBalance":            getBalance,
	"eth_getStorageAt":          getStorageAt,
	"eth_accounts":              accounts,    //mock_accounts,    //
	"eth_estimateGas":           estimateGas, //mock_estimateGas, //
	"eth_gasPrice":              gasPrice,
	"eth_sendTransaction":       sendTransaction,       //mock_sendTransaction,       //
	"eth_getTransactionReceipt": getTransactionReceipt, //mock_getTransactionReceipt, //
	"eth_getTransactionByHash":  getTransactionByHash,  //mock_getTransactionByHash,  //
	"eth_sendRawTransaction":    sendRawTransaction,
	"eth_call":                  call,
	"eth_getLogs":               getLogs,

	"eth_getBlockTransactionCountByHash":      getBlockTransactionCountByHash,
	"eth_getBlockTransactionCountByNumber":    getBlockTransactionCountByNumber,
	"eth_getTransactionByBlockHashAndIndex":   getTransactionByBlockHashAndIndex,
	"eth_getTransactionByBlockNumberAndIndex": getTransactionByBlockNumberAndIndex,
	"eth_getUncleCountByBlockHash":            getUncleCountByBlockHash,
	"eth_getUncleCountByBlockNumber":          getUncleCountByBlockNumber,
	"eth_submitWork":                          submitWork,
	"eth_submitHashrate":                      submitHashrate,
	"eth_hashrate":                            hashrate,
	"eth_getWork":                             getWork,
	"eth_protocolVersion":                     protocolVersion,
	"eth_coinbase":                            coinbase,

	"eth_sign":            sign,
	"eth_signTransaction": signTransaction,
	"eth_feeHistory":      feeHistory,
	"eth_syncing":         syncing,
	"eth_mining":          mining,

	"eth_newFilter":                   newFilter,
	"eth_newBlockFilter":              newBlockFilter,
	"eth_newPendingTransactionFilter": newPendingTransactionFilter,
	"eth_uninstallFilter":             uninstallFilter,
	"eth_getFilterChanges":            getFilterChanges,
	"eth_getFilterLogs":               getFilterLogs,

	"web3_clientVersion":     clientVersion,
	"txpool_content":         txpoolContent,
	"debug_traceTransaction": traceTransaction,

	"eth_getBlockReceipts":     getBlockReceipts,
	"eth_getHeaderByHash":      getHeaderByHash,
	"eth_getHeaderByNumber":    getHeaderByNumber,
	"eth_maxPriorityFeePerGas": maxPriorityFeePerGas,

	"eth_getProof":            getProof,
	"arn_sendRawTransactions": sendRawTransactions,
}
