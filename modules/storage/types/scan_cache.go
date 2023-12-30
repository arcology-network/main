package types

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/arcology-network/common-lib/types"
	monacoConfig "github.com/arcology-network/component-lib/config"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/core"
	evmTypes "github.com/arcology-network/evm/core/types"
	evmRlp "github.com/arcology-network/evm/rlp"
)

type Block struct {
	Height    uint64
	Time      *big.Int
	TxNumbers int
	Hash      string
	GasUsed   uint64
}
type Transaction struct {
	TxHash   string
	Height   uint64
	Time     *big.Int
	Sender   string
	SendTo   string
	Amount   *big.Int
	Status   string
	GasLimit uint64
	GasPrice *big.Int
}
type ScanCache struct {
	chainid      *big.Int
	lock         sync.RWMutex
	blockNums    int
	txNums       int
	blocks       []*Block
	transactions []*Transaction
	tabs         []uint64
}

func NewScanCache(bsize, tsize int) *ScanCache {
	return &ScanCache{
		blockNums:    bsize,
		txNums:       tsize,
		blocks:       make([]*Block, 0, bsize),
		transactions: make([]*Transaction, 0, tsize),
		chainid:      monacoConfig.MainConfig.ChainId,
		tabs:         make([]uint64, 4),
	}
}

const (
	BlockHeight = iota
	Tps
	MaxTps
	TotalTxs
)

func (sc *ScanCache) GetAllTabs() []uint64 {
	return sc.tabs
}

func (sc *ScanCache) GetTabs(style int) uint64 {
	return sc.tabs[style]
}

func (sc *ScanCache) SetTabs(style int, val uint64) {
	sc.tabs[style] = val
}

func (sc *ScanCache) BlockReceived(block *types.MonacoBlock, hash []byte, receipts map[evmCommon.Hash]*evmTypes.Receipt) error {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	tim := big.NewInt(0)

	if len(block.Headers) > 0 {
		data := block.Headers[0]
		var header evmTypes.Header
		err := evmRlp.DecodeBytes(data[1:], &header)

		if err != nil {
			return err
		}

		b := Block{
			Height:    block.Height,
			Time:      big.NewInt(int64(header.Time)),
			TxNumbers: len(block.Txs),
			//Proposer:  formatHex(header.Coinbase.Hex()),
			Hash:    fmt.Sprintf("0x%x", hash), // formatHex(fmt.Sprintf("0x%x", hash)),
			GasUsed: header.GasUsed,
		}
		sc.blocks = append(sc.blocks, &b)
		if len(sc.blocks) > sc.blockNums {
			sc.blocks = sc.blocks[1:]
		}
		tim = big.NewInt(int64(header.Time))
	}
	blocktxnum := len(block.Txs)
	rawtxs := block.Txs
	if blocktxnum >= sc.blockNums {
		rawtxs = block.Txs[blocktxnum-sc.blockNums:]
	}
	for _, tx := range rawtxs {
		transaction, err := sc.GetTransaction(tx, block.Height, tim, receipts)
		if err != nil {
			return err
		}
		sc.transactions = append(sc.transactions, transaction)
	}
	if len(sc.transactions) > sc.txNums {
		sc.transactions = sc.transactions[len(sc.transactions)-sc.txNums:]
	}
	return nil
}

func formatHex(hex string) string {
	if !strings.HasPrefix(hex, "0x") {
		hex = "0x" + hex
	}
	nhex := hex[:7] + "..." + hex[len(hex)-4:]
	return strings.ToLower(nhex)
}

func (sc *ScanCache) GetTransaction(tx []byte, height uint64, time *big.Int, receipts map[evmCommon.Hash]*evmTypes.Receipt) (*Transaction, error) {
	txType := tx[0]
	txReal := tx[1:]
	switch txType {
	case types.TxType_Eth:
		otx := new(evmTypes.Transaction)
		if err := evmRlp.DecodeBytes(txReal, otx); err != nil {
			return nil, err
		}
		txhash := types.RlpHash(otx)

		msg, err := core.TransactionToMessage(otx, evmTypes.NewEIP155Signer(sc.chainid), nil)
		if err != nil {
			return nil, err
		}
		toStr := ""
		if to := msg.To; to != nil {
			toStr = to.Hex() // formatHex(to.Hex())
		}
		status := "Failed"
		if receipts[txhash].Status == evmTypes.ReceiptStatusSuccessful {
			status = "Success"
		}

		return &Transaction{
			TxHash:   txhash.Hex(), //formatHex(txhash.Hex()),
			Height:   height,
			Time:     time,
			Sender:   msg.From.Hex(), //formatHex(msg.From().Hex()),
			SendTo:   toStr,
			Amount:   msg.Value,
			GasLimit: otx.Gas(),
			GasPrice: otx.GasPrice(),
			Status:   status,
		}, nil
	}
	return nil, errors.New("transaction type not defined")
}

func (sc *ScanCache) GetLatestBlocks() []*Block {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.blocks
}

func (sc *ScanCache) GetLatestTxs() []*Transaction {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.transactions
}

func (sc *ScanCache) GetSize() (int, int) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return len(sc.blocks), len(sc.transactions)
}
