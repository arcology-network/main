package types

import (
	"math/big"

	mainCfg "github.com/HPISTechnologies/component-lib/config"
	"github.com/HPISTechnologies/evm/common"
	"github.com/HPISTechnologies/evm/consensus"
	"github.com/HPISTechnologies/evm/core/types"
	"github.com/HPISTechnologies/evm/core/vm"
	"github.com/HPISTechnologies/evm/params"
	adaptor "github.com/HPISTechnologies/vm-adaptor/evm"
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

//var coinbase = common.BytesToAddress([]byte{100, 100, 100})
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
		GasLimit:    uint64(params.GasLimit),
		Difficulty:  new(big.Int).SetUint64(10000000),
	}
	cfg.ChainConfig.ChainID = mainCfg.MainConfig.ChainId
	cfg.Chain = new(fakeChain)
	return cfg

}
