//go:build !CI

package types

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

func newExeclog(idx int) (*types.ExecutingLogs, evmCommon.Hash) {
	logs := types.NewExecutingLogs()
	logs.Append(fmt.Sprintf("start-%v", idx), fmt.Sprintf("s-%v", idx))
	logs.Append(fmt.Sprintf("doing-%v", idx), fmt.Sprintf("s-%v", idx))
	logs.Append(fmt.Sprintf("end-%v", idx), fmt.Sprintf("s-%v", idx))
	logs.Txhash = evmCommon.BytesToHash([]byte{byte(1 + idx), 2, 3, 4, 5, 6, 7, 8, 9, 10})
	return logs, logs.Txhash
}

func TestExeclog(t *testing.T) {
	cache := NewExeclogCaches(2, 8)
	bat1 := make([]string, 3)
	hashes1 := make([]evmCommon.Hash, 3)
	logs1 := make([]*types.ExecutingLogs, 3)
	for i := range bat1 {
		log, hash := newExeclog(i)
		logs, err := log.Marshal()
		if err != nil {
			continue
		}
		bat1[i] = logs
		hashes1[i] = hash
		logs1[i] = log
	}
	cache.Save(1, bat1)

	queryResult := cache.Query(string(hashes1[1].Bytes()))
	if !reflect.DeepEqual(*queryResult, bat1[1]) {
		t.Error("cache save get Error")
		return
	}

	bat2 := make([]string, 3)
	hashes2 := make([]evmCommon.Hash, 3)
	logs2 := make([]*types.ExecutingLogs, 3)
	for i := range bat2 {
		log, hash := newExeclog(i + 3)
		logs, err := log.Marshal()
		if err != nil {
			continue
		}
		bat2[i] = logs
		hashes2[i] = hash
		logs2[i] = log
	}
	cache.Save(1, bat2)

	queryResult = cache.Query(string(hashes1[2].Bytes()))
	if !reflect.DeepEqual(*queryResult, bat1[2]) {
		t.Error("cache save get Error")
		return
	}

	queryResult = cache.Query(string(hashes2[2].Bytes()))
	if !reflect.DeepEqual(*queryResult, bat2[2]) {
		t.Error("cache save get Error")
		return
	}

	bat3 := make([]string, 3)
	hashes3 := make([]evmCommon.Hash, 3)
	logs3 := make([]*types.ExecutingLogs, 3)
	for i := range bat3 {
		log, hash := newExeclog(i + 6)
		logs, err := log.Marshal()
		if err != nil {
			continue
		}
		bat3[i] = logs
		hashes3[i] = hash
		logs3[i] = log
	}
	cache.Save(2, bat3)
	bat4 := make([]string, 3)
	hashes4 := make([]evmCommon.Hash, 3)
	logs4 := make([]*types.ExecutingLogs, 3)
	for i := range bat4 {
		log, hash := newExeclog(i + 9)
		logs, err := log.Marshal()
		if err != nil {
			continue
		}
		bat4[i] = logs
		hashes4[i] = hash
		logs4[i] = log
	}
	cache.Save(3, bat4)

	queryResult = cache.Query(string(hashes1[2].Bytes()))
	if queryResult != nil {
		t.Error("cache remove Error")
		return
	}

	queryResult = cache.Query(string(hashes3[2].Bytes()))
	if !reflect.DeepEqual(*queryResult, bat3[2]) {
		t.Error("cache save get Error")
		return
	}

	queryResult = cache.Query(string(hashes4[2].Bytes()))
	if !reflect.DeepEqual(*queryResult, bat4[2]) {
		t.Error("cache save get Error")
		return
	}

}
