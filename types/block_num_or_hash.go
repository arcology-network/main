package types

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

type RequestProof struct {
	Address        common.Address     `json:"address"`
	Keys           []common.Hash      `json:"storageKeys"`
	BlockParameter *BlockNumberOrHash `json:"blockParameter"`
}

type BlockNumberOrHash struct {
	BlockNumber *big.Int     `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash `json:"blockHash,omitempty"`
}

func ParseBlockParameter(v interface{}) (*BlockNumberOrHash, error) {
	bh := BlockNumberOrHash{}
	if str, ok := v.(string); !ok {
		return &bh, errors.New("unexpected data type given")
	} else {
		switch str {
		case "latest":
			bh.BlockNumber = big.NewInt(BlockNumberLatest)
		case "earliest":
			bh.BlockNumber = big.NewInt(BlockNumberEarliest)
		case "pending":
			bh.BlockNumber = big.NewInt(BlockNumberPending)
		case "finalized":
			bh.BlockNumber = big.NewInt(BlockNumberFinalized)
		case "safe":
			bh.BlockNumber = big.NewInt(BlockNumberSafe)
		default:
			if len(str) == 66 { //block hash
				hash := common.Hash{}
				err := hash.UnmarshalText([]byte(str))
				if err != nil {
					return &bh, err
				}
				bh.BlockHash = &hash
			} else { //block number
				if str[:2] == "0x" {
					str = str[2:]
				}
				bnn, err := strconv.ParseInt(str, 16, 0)
				if err != nil {
					return &bh, err
				}
				bh.BlockNumber = big.NewInt(bnn)
			}

		}
		return &bh, nil
	}
}

func (bnh *BlockNumberOrHash) Number() (*big.Int, bool) {
	if bnh.BlockNumber != nil {
		return bnh.BlockNumber, true
	}
	return big.NewInt(0), false
}
func (bnh *BlockNumberOrHash) Hash() (common.Hash, bool) {
	if bnh.BlockHash != nil {
		return *bnh.BlockHash, true
	}
	return common.Hash{}, false
}
