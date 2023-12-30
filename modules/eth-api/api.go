package ethapi

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	mainCfg "github.com/arcology-network/component-lib/config"
	"github.com/arcology-network/component-lib/ethrpc"
	ethereum "github.com/arcology-network/evm"
	"github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/common/hexutil"
	ethtyp "github.com/arcology-network/evm/core/types"
	"github.com/arcology-network/evm/params"
	internal "github.com/arcology-network/main/modules/eth-api/backend"
	wal "github.com/arcology-network/main/modules/eth-api/wallet"
	jsonrpc "github.com/deliveroo/jsonrpc-go"
	"github.com/rs/cors"
)

func LoadCfg(tomfile string, conf interface{}) {
	_, err := toml.DecodeFile(tomfile, conf)
	if err != nil {
		fmt.Printf("err=%v\n", err)
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

func getBlockByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[0])
	}

	isTransaction, err := ToBool(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid isTransaction  given %v", params[1])
	}

	block, err := backend.GetBlockByNumber(number, isTransaction)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return parseBlock(block, isTransaction), nil
}

func getBlockByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	isTransaction, err := ToBool(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid isTransaction  given %v", params[1])
	}

	block, err := backend.GetBlockByHash(hash, isTransaction)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return parseBlock(block, isTransaction), nil
}

func parseBlock(block *ethrpc.RPCBlock, isTransaction bool) interface{} {

	uncles := make([]string, 0)
	header := block.Header
	blockResult := map[string]interface{}{
		"uncles": uncles,

		"number":           (*hexutil.Big)(header.Number),
		"hash":             header.Hash(),
		"parentHash":       header.ParentHash,
		"nonce":            header.Nonce,
		"mixHash":          header.MixDigest,
		"sha3Uncles":       header.UncleHash,
		"logsBloom":        header.Bloom,
		"stateRoot":        header.Root,
		"miner":            header.Coinbase,
		"difficulty":       (*hexutil.Big)(header.Difficulty),
		"extraData":        hexutil.Bytes(header.Extra),
		"size":             hexutil.Uint64(header.Size()),
		"gasLimit":         hexutil.Uint64(header.GasLimit),
		"gasUsed":          hexutil.Uint64(header.GasUsed),
		"timestamp":        hexutil.Uint64(header.Time),
		"transactionsRoot": header.TxHash,
		"receiptsRoot":     header.ReceiptHash,
		"totalDifficulty":  (*hexutil.Big)(header.Difficulty),
		// "transactions":     transactions,
	}

	if isTransaction {
		transactions := make([]*ethrpc.RPCTransaction, len(block.Transactions))
		for i := range block.Transactions {
			transactions[i] = AttachChainId(block.Transactions[i].(*ethrpc.RPCTransaction), options.ChainID)
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
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[1])
	}

	nonce, err := backend.GetTransactionCount(address, number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(nonce), nil
}

func getCode(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[1])
	}

	code, err := backend.GetCode(address, number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("0x%x", code), nil
}

func getBalance(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	number, err := ToBlockNumber(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[1])
	}

	balance, err := backend.GetBalance(address, number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(balance), nil
}

func getStorageAt(ctx context.Context, params []interface{}) (interface{}, error) {
	address, err := ToAddress(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}

	key, err := ToHash(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid key given %v", params[1])
	}

	number, err := ToBlockNumber(params[2])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid block number given %v", params[2])
	}

	value, err := backend.GetStorageAt(address, key.Hex(), number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return "0x" + hex.EncodeToString(value), nil
}

func accounts(ctx context.Context) (interface{}, error) {
	return wallet.Accounts(), nil
}

func estimateGas(ctx context.Context, params []interface{}) (interface{}, error) {
	// msg, err := ToCallMsg(params[0], true)
	// if err != nil {
	// 	fmt.Printf("*************invalid call msg given %v\n", params[0])
	// 	return nil, jsonrpc.InvalidParams("invalid call msg given %v", params[0])
	// }

	//gas, err := backend.EstimateGas(msg)
	gas, err := backend.EstimateGas(ethereum.CallMsg{})
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(gas), nil
}

func gasPrice(ctx context.Context) (interface{}, error) {
	gp, err := backend.GasPrice()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(gp), nil
}

func sendTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	tx, err := ToSendTxArgs(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid transaction given %v", params[0])
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

func getTransactionReceipt(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
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
				return nil, jsonrpc.InternalError(err)
			}
			time.Sleep(time.Duration(time.Second * 1))
		} else {
			break
		}
	}

	tx, err := backend.GetTransactionByHash(hash)
	if err != nil {
		fmt.Printf("<<<<<<<<<<<<<<<<getTransactionByHash err:%v\n", err)
		return nil, jsonrpc.InternalError(err)
	}

	fields := map[string]interface{}{
		"blockHash":         receipt.BlockHash,
		"blockNumber":       hexutil.Uint64(receipt.BlockNumber.Uint64()),
		"transactionHash":   hash,
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

	return fields, nil
}

func getTransactionByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	tx, err := backend.GetTransactionByHash(hash)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}

	return ToTransactionResponse(tx, options.ChainID), nil
}

func sendRawTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	bytes, err := ToBytes(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid raw transaction given %v", params[0])
	}

	hash, err := backend.SendRawTransaction(bytes)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return hash.Hex(), nil
}

func call(ctx context.Context, params []interface{}) (interface{}, error) {
	msg, err := ToCallMsg(params[0], true)
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid call msg given %v", params[0])
	}
	if msg.Gas == 0 {
		msg.Gas = math.MaxUint32
	}
	if msg.GasPrice == nil {
		msg.GasPrice = big.NewInt(0xff)
	}
	if msg.Value == nil {
		msg.Value = big.NewInt(0)
	}
	ret, err := backend.Call(msg)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return "0x" + hex.EncodeToString(ret), nil
}

func getLogs(ctx context.Context, params []interface{}) (interface{}, error) {
	filter, err := ToFilter(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}

	logs, err := backend.GetLogs(filter)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return logs, nil
}

func getBlockTransactionCountByHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	txsNum, err := backend.GetBlockTransactionCountByHash(hash)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(txsNum), nil
}

func getBlockTransactionCountByNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid number given %v", params[0])
	}

	txsNum, err := backend.GetBlockTransactionCountByNumber(number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(txsNum), nil
}

func getTransactionByBlockHashAndIndex(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	index, err := ToBlockIndex(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid index given %v", params[1])
	}

	tx, err := backend.GetTransactionByBlockHashAndIndex(hash, index)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return ToTransactionResponse(tx, options.ChainID), nil
}

func getTransactionByBlockNumberAndIndex(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid number given %v", params[0])
	}
	index, err := ToBlockIndex(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid index given %v", params[1])
	}

	tx, err := backend.GetTransactionByBlockNumberAndIndex(number, index)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return ToTransactionResponse(tx, options.ChainID), nil
}

func getUncleCountByBlockHash(ctx context.Context, params []interface{}) (interface{}, error) {
	hash, err := ToHash(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid hash given %v", params[0])
	}

	txsNum, err := backend.GetUncleCountByBlockHash(hash)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return NumberToHex(txsNum), nil
}

func getUncleCountByBlockNumber(ctx context.Context, params []interface{}) (interface{}, error) {
	number, err := ToBlockNumber(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid number given %v", params[0])
	}

	txsNum, err := backend.GetUncleCountByBlockNumber(number)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
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
		return nil, jsonrpc.InvalidParams("invalid address given %v", params[0])
	}
	txdata, err := ToBytes(params[1])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid bytes given %v", params[0])
	}
	retData, err := wallet.Sign(address, txdata)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%x", retData), nil
}

func signTransaction(ctx context.Context, params []interface{}) (interface{}, error) {
	tx, err := ToSendTxArgs(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid transaction given %v", params[0])
	}
	rawTx, err := wallet.SignTx(0, tx.Nonce, tx.To, tx.Value, tx.Gas, tx.GasPrice, tx.Data)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%x", rawTx), nil
}

func feeHistory(ctx context.Context) (interface{}, error) {
	return ethrpc.FeeHistoryResult{}, nil
}

func syncing(ctx context.Context) (interface{}, error) {
	ok, err := backend.Syncing()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%v", ok), nil
}

func mining(ctx context.Context) (interface{}, error) {
	ok, err := backend.Proposer()
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return fmt.Sprintf("%v", ok), nil
}

func newFilter(ctx context.Context, params []interface{}) (interface{}, error) {
	filter, err := ToFilter(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
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
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
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
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}
	results, err := backend.GetFilterChanges(id)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
	}
	return results, nil
}

func getFilterLogs(ctx context.Context, params []interface{}) (interface{}, error) {
	id, err := ToID(params[0])
	if err != nil {
		return nil, jsonrpc.InvalidParams("invalid filter given %v", params[0])
	}

	logs, err := backend.GetFilterLogs(id)
	if err != nil {
		return nil, jsonrpc.InternalError(err)
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

func startJsonRpc() {
	filters := internal.NewFilters()
	options.ChainID = mainCfg.MainConfig.ChainId.Uint64()
	fmt.Println(options)

	if options.ChainID == 0 {
		options.ChainID = params.MainnetChainConfig.ChainID.Uint64()
	}
	privateKeys := LoadKeys(options.KeyFile)

	logFile, err := os.OpenFile("./rpc.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("open log file failed, err:", err)
		return
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Lmicroseconds | log.Ldate)

	server := jsonrpc.New()
	server.Use(func(next jsonrpc.Next) jsonrpc.Next {
		return func(ctx context.Context, params interface{}) (interface{}, error) {
			// method := jsonrpc.MethodFromContext(ctx)
			// fmt.Printf("***********************************************method: %v \t params:%v \n", method, params)
			return next(ctx, params)
		}
	})
	server.Register(jsonrpc.Methods{
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
	})

	if options.Debug {
		backend = internal.NewEthereumAPIMock(new(big.Int).SetUint64(options.ChainID))
	} else {
		backend = internal.NewMonaco(filters)
	}

	wallet = wal.NewWallet(new(big.Int).SetUint64(options.ChainID), privateKeys)

	c := cors.AllowAll()
	go http.ListenAndServe(fmt.Sprintf(":%d", options.Port), c.Handler(server))
}
