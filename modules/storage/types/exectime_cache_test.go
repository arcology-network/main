package types

import (
	"reflect"
	"testing"

	"github.com/arcology-network/common-lib/types"
)

func TestExectime(t *testing.T) {
	cache := NewExectimeCaches(2)
	exectime1 := types.StatisticalInformation{
		Key:      "exectime",
		Value:    "ok",
		TimeUsed: 100,
	}
	cache.Save(1, &exectime1)
	exectime2 := types.StatisticalInformation{
		Key:      "exectime",
		Value:    "ok",
		TimeUsed: 200,
	}
	cache.Save(2, &exectime2)

	queryResult := cache.Query(1)
	if !reflect.DeepEqual(*queryResult, exectime1) {
		t.Error("cache save get Error")
		return
	}
	exectime3 := types.StatisticalInformation{
		Key:      "exectime",
		Value:    "ok",
		TimeUsed: 300,
	}
	cache.Save(3, &exectime3)

	queryResult = cache.Query(1)
	if queryResult != nil {
		t.Error("cache remove Error")
		return
	}

	queryResult = cache.Query(2)
	if !reflect.DeepEqual(*queryResult, exectime2) {
		t.Error("cache save get Error")
		return
	}

}
