package types

import (
	"fmt"
	"reflect"
	"testing"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

func Test_ExecutorResponses_EncodingAndDeconing(t *testing.T) {
	// defs := []*DeferredCall{
	// 	{
	// 		DeferID:         "123123",
	// 		ContractAddress: Address("defcalll123122121212"),
	// 		Signature:       "defcall call()",
	// 	},
	// 	{
	// 		DeferID:         "345678",
	// 		ContractAddress: Address("defcalll123122121452"),
	// 		Signature:       "defcall call(s int)",
	// 	},
	// 	{
	// 		DeferID:         "9101112",
	// 		ContractAddress: Address("defcalll123333332152"),
	// 		Signature:       "defcall call(s uint64)",
	// 	},
	// }

	// defcalls := make([]*DeferredCall, 3)
	// defcalls[0] = defs[0]
	// defcalls[1] = defs[1]
	// defcalls[2] = defs[2]

	hashes := []ethCommon.Hash{
		ethCommon.BytesToHash([]byte{1, 2, 3}),
		ethCommon.BytesToHash([]byte{4, 5, 6}),
		ethCommon.BytesToHash([]byte{7, 8, 9}),
	}
	statusList := []uint64{1, 1, 0}
	gasUsedList := []uint64{10, 11, 12}

	// spawnedTxs := []ethCommon.Hash{
	// 	ethCommon.BytesToHash([]byte{1, 1, 1}),
	// 	ethCommon.BytesToHash([]byte{2, 2, 2}),
	// }

	// relationKeys := []ethCommon.Hash{
	// 	ethCommon.BytesToHash([]byte{3, 3, 3}),
	// 	ethCommon.BytesToHash([]byte{4, 4, 4}),
	// }

	// relationSizes := []uint64{
	// 	uint64(2),
	// 	uint64(2),
	// }

	// relationValues := []ethCommon.Hash{
	// 	ethCommon.BytesToHash([]byte{5, 5, 5}),
	// 	ethCommon.BytesToHash([]byte{6, 6, 6}),
	// 	ethCommon.BytesToHash([]byte{7, 7, 7}),
	// 	ethCommon.BytesToHash([]byte{8, 8, 8}),
	// }

	er := ExecutorResponses{
		// DfCalls:        defcalls,
		HashList:    hashes,
		StatusList:  statusList,
		GasUsedList: gasUsedList,
		// SpawnedTxs:     spawnedTxs,
		// RelationKeys:   relationKeys,
		// RelationSizes:  relationSizes,
		// RelationValues: relationValues,
	}

	data, err := er.GobEncode()
	if err != nil {
		fmt.Printf("ExecutorResponses encode err=%v\n", err)
		return
	}
	fmt.Printf("ExecutorResponses encode result=%v\n", data)

	erResult := ExecutorResponses{}

	err = erResult.GobDecode(data)
	if err != nil {
		fmt.Printf(" ExecutorResponses.GobDecode err=%v\n", err)
		return

	}

	fmt.Printf(" ExecutorResponses.GobDecode result=%v\n", erResult)

	// for i, def := range er.DfCalls {
	// 	if !reflect.DeepEqual(*def, *erResult.DfCalls[i]) {
	// 		t.Error("ExecutorResponses encode docode err in defcalls!", *def, *erResult.DfCalls[i], i)
	// 	}
	// }
	if !reflect.DeepEqual(er.GasUsedList, erResult.GasUsedList) {
		t.Error("ExecutorResponses encode docode err in GasUsedList!", er.GasUsedList, erResult.GasUsedList)
	}
	if !reflect.DeepEqual(er.HashList, erResult.HashList) {
		t.Error("ExecutorResponses encode docode err in HashList!", er.HashList, erResult.HashList)
	}
	if !reflect.DeepEqual(er.StatusList, erResult.StatusList) {
		t.Error("ExecutorResponses encode docode err in StatusList!", er.StatusList, erResult.StatusList)
	}

	// if !reflect.DeepEqual(er.RelationKeys, erResult.RelationKeys) {
	// 	t.Error("ExecutorResponses encode docode err in RelationKeys!", er.RelationKeys, erResult.RelationKeys)
	// }
	// if !reflect.DeepEqual(er.RelationSizes, erResult.RelationSizes) {
	// 	t.Error("ExecutorResponses encode docode err in RelationSizes!", er.RelationSizes, erResult.RelationSizes)
	// }
	// if !reflect.DeepEqual(er.RelationValues, erResult.RelationValues) {
	// 	t.Error("ExecutorResponses encode docode err in RelationValues!", er.RelationValues, erResult.RelationValues)
	// }
	// if !reflect.DeepEqual(er.SpawnedTxs, erResult.SpawnedTxs) {
	// 	t.Error("ExecutorResponses encode docode err in SpawnedTxs!", er.SpawnedTxs, erResult.SpawnedTxs)
	// }
}
