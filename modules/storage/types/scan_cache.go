package types

import (
	"errors"
	"math/big"
	"strings"
	"sync"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethRlp "github.com/HPISTechnologies/3rd-party/eth/rlp"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/types"
	monacoConfig "github.com/HPISTechnologies/component-lib/config"
)

type Block struct {
	Height    uint64
	Time      *big.Int
	TxNumbers int
	Proposer  string
	GasUsed   uint64
}
type Transaction struct {
	TxHash   string
	Height   uint64
	Time     *big.Int
	From     string
	To       string
	Value    *big.Int
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
}

func NewScanCache(bsize, tsize int) *ScanCache {
	return &ScanCache{
		blockNums:    bsize,
		txNums:       tsize,
		blocks:       make([]*Block, 0, bsize),
		transactions: make([]*Transaction, 0, tsize),
		chainid:      monacoConfig.MainConfig.ChainId,
	}
}
func (sc *ScanCache) BlockReceived(block *types.MonacoBlock) error {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	tim := big.NewInt(0)

	if len(block.Headers) > 0 {
		data := block.Headers[0]
		var header ethTypes.Header
		err := ethRlp.DecodeBytes(data[1:], &header)

		if err != nil {
			return err
		}

		b := Block{
			Height:    block.Height,
			Time:      header.Time,
			TxNumbers: len(block.Txs),
			Proposer:  formatHex(header.Coinbase.Hex()),
			GasUsed:   header.GasUsed,
		}
		sc.blocks = append(sc.blocks, &b)
		if len(sc.blocks) > sc.blockNums {
			sc.blocks = sc.blocks[1:]
		}
		tim = header.Time
	}
	blocktxnum := len(block.Txs)
	rawtxs := block.Txs
	if blocktxnum >= sc.blockNums {
		rawtxs = block.Txs[blocktxnum-sc.blockNums:]
	}
	for _, tx := range rawtxs {
		transaction, err := sc.GetTransaction(tx, block.Height, tim)
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
func (sc *ScanCache) GetTransaction(tx []byte, height uint64, time *big.Int) (*Transaction, error) {
	txType := tx[0]
	txReal := tx[1:]
	switch txType {
	case types.TxType_Eth:
		otx := new(ethTypes.Transaction)
		if err := ethRlp.DecodeBytes(txReal, otx); err != nil {
			return nil, err
		}
		txhash := ethCommon.RlpHash(otx)
		msg, err := otx.AsMessage(ethTypes.NewEIP155Signer(sc.chainid))
		if err != nil {
			return nil, err
		}
		toStr := ""
		if to := msg.To(); to != nil {
			toStr = formatHex(to.Hex())
		}
		return &Transaction{
			TxHash:   formatHex(txhash.Hex()),
			Height:   height,
			Time:     time,
			From:     formatHex(msg.From().Hex()),
			To:       toStr,
			Value:    msg.Value(),
			GasLimit: otx.Gas(),
			GasPrice: otx.GasPrice(),
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
