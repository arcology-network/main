package types

import (
	"strings"

	"github.com/arcology-network/vm-adaptor/eth"
	evmcommon "github.com/ethereum/go-ethereum/common"
)

const (
	nthread = 4
)

var connector *eth.EthPathBuilder //eth.EthCCurlConnector

func init() {
	connector = &eth.EthPathBuilder{}
}

func getBalancePath(addr string) string {
	return connector.BalancePath(evmcommon.HexToAddress(addr))
	//return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Balance)
}
func getNoncePath(addr string) string {
	return connector.NoncePath(evmcommon.HexToAddress(addr))
	// return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Nonce)
}
func getCodePath(addr string) string {
	return connector.CodePath(evmcommon.HexToAddress(addr))
	// return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Code)
}

func getStorageKeyPath(addr, key string) string {
	// paths, _, _ := concurrenturl.NewPlatform().Builtin(BASE_URL, addr)
	if !strings.HasPrefix(key, "0x") {
		key = "0x" + key
	}

	return connector.StorageRootPath(evmcommon.HexToAddress(addr)) + key
	// return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Native) + key
}
