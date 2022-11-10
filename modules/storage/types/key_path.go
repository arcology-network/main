package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	urlcommon "github.com/arcology-network/concurrenturl/v2/common"
)

const (
	nthread = 4
)

var BASE_URL string

func init() {
	BASE_URL = urlcommon.NewPlatform().Eth10()
}

func getBalancePath(addr string) string {
	paths, _, _ := urlcommon.NewPlatform().Builtin(BASE_URL, addr)
	return paths[3]
}
func getNoncePath(addr string) string {
	paths, _, _ := urlcommon.NewPlatform().Builtin(BASE_URL, addr)
	return paths[2]
}
func getCodePath(addr string) string {
	paths, _, _ := urlcommon.NewPlatform().Builtin(BASE_URL, addr)
	return paths[1]
}

func getStorageKeyPath(addr, key string) string {
	paths, _, _ := urlcommon.NewPlatform().Builtin(BASE_URL, addr)
	if !strings.HasPrefix(key, "0x") {
		key = "0x" + key
	}
	return paths[7] + key
}

func getContainerValuePath(addr, id string, key interface{}) string {
	paths, _, _ := urlcommon.NewPlatform().Builtin(BASE_URL, addr)
	return fmt.Sprintf("%s%v", paths[6]+id+"/", key)
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
