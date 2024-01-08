package pool

import (
	"fmt"
	"math/big"

	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ReapPair struct {

	//config
	MaxReap    int
	ChainID    *big.Int
	Signer     evmTypes.Signer
	SignerType uint8

	//request process
	ReapCommand bool
	Opcaches    *[]*types.StandardTransaction
	Withdrawals []*evmTypes.Withdrawal
	// OpMsgs      []*types.StandardMessage
	Oplist   []*evmCommon.Hash
	ReapSize int

	//result process
	MBlock   *types.MonacoBlock
	Receipts *[]*evmTypes.Receipt
}

func NewReapPair(maxReap int, chainId *big.Int) *ReapPair {
	return &ReapPair{
		ReapCommand: false,
		ReapSize:    maxReap,
		MaxReap:     maxReap,
		ChainID:     chainId,
		Signer:      evmTypes.NewLondonSigner(chainId),
		SignerType:  types.Signer_London,
	}
}

func (r *ReapPair) ChangeSigner(signerType uint8) {
	r.Signer = types.GetSigner(signerType, r.ChainID)
}

func (r *ReapPair) Reset() {
	r.ReapCommand = false
	r.Opcaches = nil
	// r.OpMsgs = nil
	r.Oplist = nil
	r.Withdrawals = nil
	r.ReapSize = r.MaxReap

	r.MBlock = nil
	r.Receipts = nil
}
func (r *ReapPair) Check() bool {
	if !r.ReapCommand || r.Opcaches == nil {
		return false
	}
	return true
}

func (r *ReapPair) AddReapCommand() bool {
	r.ReapCommand = true
	return r.Check()
}
func (r *ReapPair) AddOpCommand(txs []*types.StandardTransaction, withdrawals []*evmTypes.Withdrawal) bool {

	msgs := make([]*types.StandardTransaction, 0, len(txs))
	list := make([]*evmCommon.Hash, 0, len(txs))
	for _, tx := range txs {
		if tx.UnSign(r.Signer) != nil {
			panic("ReapPair.AddOpCommand unsign err")
		}
		tx.Signer = r.SignerType
		msgs = append(msgs, tx)
		list = append(list, &tx.TxHash)
	}
	r.Opcaches = &msgs
	r.Oplist = list
	r.Withdrawals = withdrawals

	r.ReapSize = r.ReapSize - len(list)
	if r.ReapSize < 0 {
		r.ReapSize = 0
	}

	return r.Check()
}

func (r *ReapPair) AppendList(list []*evmCommon.Hash) []*evmCommon.Hash {
	return append(r.Oplist, list...)
}

func (r *ReapPair) ClipReapList(list []evmCommon.Hash) []evmCommon.Hash {
	return list[len(r.Oplist):]
}

func (r *ReapPair) ReapEnd(reaped []*types.StandardTransaction) ([]*types.StandardTransaction, []*evmTypes.Transaction, [][]byte) {
	for _, msg := range reaped {
		if msg.NativeMessage == nil {
			continue
		}
		if msg.Signer != r.SignerType {
			continue
		}
		(*r.Opcaches) = append((*r.Opcaches), msg)
	}
	Transactions := make([]*evmTypes.Transaction, 0, len(*r.Opcaches))
	txs := make([][]byte, 0, len(*r.Opcaches))
	for _, tx := range *r.Opcaches {
		Transactions = append(Transactions, tx.NativeTransaction)
		txs = append(txs, tx.TxRawData)
	}
	return *r.Opcaches, Transactions, txs
}

// *****************************************************************************
func (r *ReapPair) AddBlock(block *types.MonacoBlock) (bool, *types.BlockResult) {
	r.MBlock = block
	return r.Calculate()
}
func (r *ReapPair) AddReceipts(receipts []*evmTypes.Receipt) (bool, *types.BlockResult) {
	r.Receipts = &receipts
	return r.Calculate()
}

func (r *ReapPair) Calculate() (bool, *types.BlockResult) {
	if r.MBlock == nil || r.Receipts == nil {
		return false, nil
	}

	data := r.MBlock.Headers[0]
	var header evmTypes.Header
	err := header.UnmarshalJSON(data[1:])
	if err != nil {
		return false, nil
	}
	txs := make([]*evmTypes.Transaction, len(*r.Opcaches))
	for i, tx := range *r.Opcaches {
		txs[i] = tx.NativeTransaction
	}
	// block := evmTypes.NewBlockWithWithdrawals(&header, txs, []*evmTypes.Header{}, *r.Receipts, r.Withdrawals, trie.NewStackTrie(nil))
	block := evmTypes.NewBlockWithHeader(&header)
	block.AttachBody(txs, []*evmTypes.Header{}, r.Withdrawals)
	fmt.Printf(">>>>>>>>>>>>>>>>>>>main/modules/pool/reap_pair.go>>>>>>>>>>>blockhash:%x,number:%v,mblockhash:%x\n", block.Hash(), block.Number(), r.MBlock.Hash())
	return true, &types.BlockResult{
		Block: block,
		Fees:  totalFees(block, *r.Receipts),
	}
}

// totalFees computes total consumed miner fees in Wei. Block transactions and receipts have to have the same order.
func totalFees(block *evmTypes.Block, receipts []*evmTypes.Receipt) *big.Int {
	feesWei := new(big.Int)
	for i, tx := range block.Transactions() {
		minerFee, _ := tx.EffectiveGasTip(block.BaseFee())
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(receipts[i].GasUsed), minerFee))
	}
	return feesWei
}
