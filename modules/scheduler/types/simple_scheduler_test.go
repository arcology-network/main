package types

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"testing"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/signature"
	"github.com/HPISTechnologies/common-lib/types"
)

func TestSchedulerPerformance(t *testing.T) {
	stdMsgs := make([]*types.StandardMessage, 10)

	for i := 0; i < len(stdMsgs); i++ {
		to := ethCommon.BytesToAddress([]byte{11, 8, 9, 10})
		bytes := sha256.Sum256([]byte(fmt.Sprint(i)))
		ethMsg := ethTypes.NewMessage(
			ethCommon.BytesToAddress(bytes[:]),
			&to,
			uint64(10),
			big.NewInt(12000000),
			uint64(22),
			big.NewInt(34),
			make([]byte, 128),
			false,
		)

		stdMsgs[i] = &types.StandardMessage{
			Source: 1,
			TxHash: bytes,
			Native: &ethMsg,
		}

	}

	concurrencyLookup := signature.GetParallelFuncMap()
	t0 := time.Now()
	schedule := NewExecutionSchedule(stdMsgs, &concurrencyLookup, nil)
	schedule.GetNextBatch()

	fmt.Println("Total NewExecutionSchedule:", time.Since(t0))
}

func TestUint64Performance(t *testing.T) {
	t0 := time.Now()
	hashes := make([]ethCommon.Hash, 500000)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = sha256.Sum256([]byte(fmt.Sprint(i)))
	}
	fmt.Println("sort.hashes:", time.Since(t0))

	t0 = time.Now()
	nums := make([]uint64, len(hashes))
	for i := 0; i < len(nums); i++ {
		nums[i] = binary.LittleEndian.Uint64(hashes[i][:8])
	}
	fmt.Println("Uint64:", time.Since(t0))

	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
}

func TestBigint(t *testing.T) {
	t0 := time.Now()
	var fee big.Int
	for i := 0; i < 100000; i++ {
		big.NewInt(0).Mul(big.NewInt(int64(i)), big.NewInt(int64(i)))
	}
	fmt.Println(fee)
	fmt.Println(time.Since(t0))
}
