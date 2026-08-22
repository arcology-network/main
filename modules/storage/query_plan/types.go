package queryplan

import (
	"math/big"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

const (
	QueryKey_TxHash         = "_txhash"
	QueryKey_BlockHash      = "_blockhash"
	QueryKey_BlockHashExist = "_blockhashexist"
	QueryKey_Hashes         = "_hashes"
	QueryKey_Receipt        = "_receipt"
	QueryKey_Receipts       = "_receipts"
	QueryKey_Height         = "_height"
	QueryKey_Block          = "_block"
	QueryKey_ArcologyBlock  = "_arcologyblock"
	QueryKey_Position       = "_position"
	QueryKey_AccountAddress = "_accountAddress"
	QueryKey_Address        = "_address"
	QueryKey_StorageKey     = "_storagekey"
	QueryKey_Storage        = "_storage"

	QueryKey_Balance          = "_balance"
	QueryKey_BlockParams      = "_blockParams"
	QueryKey_Code             = "_code"
	QueryKey_Nonce            = "_nonce"
	QueryKey_Transaction      = "_transaction"
	QueryKey_TransactionCount = "_transactionCount"

	QueryKey_ChainID        = "_chainID"
	QueryKey_SignerType     = "_signerType"
	QueryKey_RpcTransaction = "_rpcTransaction"
	QueryKey_RpcBlock       = "_rpcBlock"
	QueryKey_Fulltx         = "_fulltx"
	QueryKey_OnlyHeader     = "_onlyHeader"
	QueryKey_Transactions   = "_transactions"

	QueryKey_RpcBlockResult = "_rpcBlockResult"
	QueryKey_IdxInBlock     = "_idxInBlock"
	QueryKey_StateRoot      = "_stateRoot"
	QueryKey_RequestId      = "_requestId"
)

type QueryParamTransactionByPosition struct {
	ChainId    *big.Int
	Position   *mstypes.Position
	BlockHash  evmCommon.Hash
	SignerType uint8
}

type QueryParamRpcBlock struct {
	Height     *big.Int
	Fulltx     bool
	OnlyHeader bool
	ChainId    *big.Int
}
