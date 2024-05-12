package types

import (
	"strings"

	adaptorcommon "github.com/arcology-network/evm-adaptor/pathbuilder"
	evmcommon "github.com/ethereum/go-ethereum/common"
)

const (
	nthread = 4
)

var connector *adaptorcommon.EthPathBuilder

func init() {
	connector = &adaptorcommon.EthPathBuilder{}
}

func getBalancePath(addr string) string {
	return connector.BalancePath(evmcommon.HexToAddress(addr))
}
func getNoncePath(addr string) string {
	return connector.NoncePath(evmcommon.HexToAddress(addr))
}
func getCodePath(addr string) string {
	return connector.CodePath(evmcommon.HexToAddress(addr))
}

func getStorageKeyPath(addr, key string) string {
	if !strings.HasPrefix(key, "0x") {
		key = "0x" + key
	}

	return connector.StorageRootPath(evmcommon.HexToAddress(addr)) + key
}
