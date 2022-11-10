package types

import (
	"math"
	"math/big"

	mainCfg "github.com/arcology-network/component-lib/config"
	"github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/consensus"
	"github.com/arcology-network/evm/core/types"
	"github.com/arcology-network/evm/core/vm"
	"github.com/arcology-network/evm/params"
	adaptor "github.com/arcology-network/vm-adaptor/evm"
)

// fakeChain implements the ChainContext interface.
type fakeChain struct {
}

func (chain *fakeChain) GetHeader(common.Hash, uint64) *types.Header {
	return &types.Header{}
}

func (chain *fakeChain) Engine() consensus.Engine {
	return nil
}

// var coinbase = common.BytesToAddress([]byte{100, 100, 100})
var coinbase = common.HexToAddress("0x3d361736e7c94ee64f74c57a82b2af7ee17c2bf1")

func MainConfig() *adaptor.Config {
	vmConfig := vm.Config{}
	cfg := &adaptor.Config{
		ChainConfig: params.MainnetChainConfig,
		VMConfig:    &vmConfig,
		BlockNumber: new(big.Int).SetUint64(10000000),
		ParentHash:  common.Hash{},
		Time:        new(big.Int).SetUint64(10000000),
		Coinbase:    &coinbase,
		GasLimit:    math.MaxUint64,
		Difficulty:  new(big.Int).SetUint64(10000000),
	}
	cfg.ChainConfig.ChainID = mainCfg.MainConfig.ChainId
	cfg.Chain = new(fakeChain)
	return cfg

}
