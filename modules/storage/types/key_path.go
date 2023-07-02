package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/arcology-network/concurrenturl"
)

const (
	nthread = 4
)

var BASE_URL string

func init() {
	BASE_URL = concurrenturl.NewPlatform().Eth10()
}

func getBalancePath(addr string) string {
	return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Balance)
}
func getNoncePath(addr string) string {
	return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Nonce)
}
func getCodePath(addr string) string {
	return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Code)
}

func getStorageKeyPath(addr, key string) string {
	// paths, _, _ := concurrenturl.NewPlatform().Builtin(BASE_URL, addr)
	if !strings.HasPrefix(key, "0x") {
		key = "0x" + key
	}
	return concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Native) + key
}

func getContainerValuePath(addr, id string, key interface{}) string {
	// paths, _, _ := concurrenturl.NewPlatform().Builtin(BASE_URL, addr)
	return fmt.Sprintf("%s%v", concurrenturl.NewPlatform().Builtins(addr, concurrenturl.Idx_PathKey_Container)+id+"/", key)
}
func getContainerStoredKey(key []byte) string {
	return "$" + hex.EncodeToString(key)
}
func getContainerArrayPath(addr, id string, idx int) string {
	return getContainerValuePath(addr, id, idx)
}
func getContainerMapPath(addr, id string, key []byte) string {
	return getContainerValuePath(addr, id, getContainerStoredKey(key))
}
func getContainerQueuePath(addr, id string, key []byte) string {
	return getContainerValuePath(addr, id, getContainerStoredKey(key))
}
