package pool

import (
	"math/big"

	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type OpAdaptor struct {

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

func NewOpAdaptor(maxReap int, chainId *big.Int) *OpAdaptor {
	return &OpAdaptor{
		ReapCommand: false,
		ReapSize:    maxReap,
		MaxReap:     maxReap,
		ChainID:     chainId,
		Signer:      evmTypes.NewLondonSigner(chainId),
		SignerType:  types.Signer_London,
	}
}

func (oa *OpAdaptor) ChangeSigner(signerType uint8) {
	oa.Signer = types.GetSigner(signerType, oa.ChainID)
}

func (oa *OpAdaptor) Reset() {
	oa.ReapCommand = false
	oa.Opcaches = nil
	// r.OpMsgs = nil
	oa.Oplist = nil
	oa.Withdrawals = nil
	oa.ReapSize = oa.MaxReap

	oa.MBlock = nil
	oa.Receipts = nil
}
func (oa *OpAdaptor) Check() bool {
	if !oa.ReapCommand || oa.Opcaches == nil {
		return false
	}
	return true
}

func (oa *OpAdaptor) AddReapCommand() bool {
	oa.ReapCommand = true
	return oa.Check()
}
func (oa *OpAdaptor) AddOpCommand(txs []*types.StandardTransaction, withdrawals []*evmTypes.Withdrawal) bool {

	msgs := make([]*types.StandardTransaction, 0, len(txs))
	list := make([]*evmCommon.Hash, 0, len(txs))
	for _, tx := range txs {
		if tx.UnSign(oa.Signer) != nil {
			panic("ReapPair.AddOpCommand unsign err")
		}
		tx.Signer = oa.SignerType
		msgs = append(msgs, tx)
		list = append(list, &tx.TxHash)
	}
	oa.Opcaches = &msgs
	oa.Oplist = list
	oa.Withdrawals = withdrawals

	oa.ReapSize = oa.ReapSize - len(list)
	if oa.ReapSize < 0 {
		oa.ReapSize = 0
	}

	return oa.Check()
}

func (oa *OpAdaptor) AppendList(list []*evmCommon.Hash) []*evmCommon.Hash {
	return append(oa.Oplist, list...)
}

func (oa *OpAdaptor) ClipReapList(list []evmCommon.Hash) []evmCommon.Hash {
	return list[len(oa.Oplist):]
}

func (oa *OpAdaptor) ReapEnd(reaped []*types.StandardTransaction) ([]*types.StandardTransaction, []*evmTypes.Transaction, [][]byte) {
	for _, msg := range reaped {
		if msg.NativeMessage == nil {
			continue
		}
		if msg.Signer != oa.SignerType {
			continue
		}
		(*oa.Opcaches) = append((*oa.Opcaches), msg)
	}
	Transactions := make([]*evmTypes.Transaction, 0, len(*oa.Opcaches))
	txs := make([][]byte, 0, len(*oa.Opcaches))
	for _, tx := range *oa.Opcaches {
		Transactions = append(Transactions, tx.NativeTransaction)
		txs = append(txs, tx.TxRawData)
	}
	return *oa.Opcaches, Transactions, txs
}

// *****************************************************************************
func (oa *OpAdaptor) AddBlock(block *types.MonacoBlock) (bool, *types.BlockResult) {
	oa.MBlock = block
	return oa.Calculate()
}
func (oa *OpAdaptor) AddReceipts(receipts []*evmTypes.Receipt) (bool, *types.BlockResult) {
	oa.Receipts = &receipts
	return oa.Calculate()
}

func (oa *OpAdaptor) Calculate() (bool, *types.BlockResult) {
	if oa.MBlock == nil || oa.Receipts == nil {
		return false, nil
	}

	data := oa.MBlock.Headers[0]
	var header evmTypes.Header
	err := header.UnmarshalJSON(data[1:])
	if err != nil {
		return false, nil
	}
	txs := make([]*evmTypes.Transaction, len(*oa.Opcaches))
	for i, tx := range *oa.Opcaches {
		txs[i] = tx.NativeTransaction
	}
	// block := evmTypes.NewBlockWithWithdrawals(&header, txs, []*evmTypes.Header{}, *r.Receipts, r.Withdrawals, trie.NewStackTrie(nil))
	block := evmTypes.NewBlockWithHeader(&header)
	block.AttachBody(txs, []*evmTypes.Header{}, oa.Withdrawals)
	return true, &types.BlockResult{
		Block: block,
		Fees:  totalFees(block, *oa.Receipts),
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
