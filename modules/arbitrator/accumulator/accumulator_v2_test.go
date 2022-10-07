package accumulator_test

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/HPISTechnologies/main/modules/arbitrator/accumulator"
	"github.com/HPISTechnologies/main/modules/arbitrator/types"
)

func TestAccumulatorV2Basic(t *testing.T) {
	transitions := []*types.BalanceTransition{
		{"alice/balance", 0, new(big.Int).SetInt64(100), new(big.Int).SetInt64(-10)},
		{"alice/balance", 1, new(big.Int).SetInt64(100), new(big.Int).SetInt64(-110)},
		{"bob/balance", 2, new(big.Int).SetInt64(200), new(big.Int).SetInt64(-220)},
		{"bob/balance", 3, new(big.Int).SetInt64(200), new(big.Int).SetInt64(-20)},
	}

	conflicts := accumulator.Check(transitions, 4)
	t.Log(conflicts)
}

func BenchmarkAccumulatorV2(b *testing.B) {
	NT := 1000000
	NA := 1000
	addresses := make([]string, NA)
	for i := 0; i < NA; i++ {
		addresses[i] = RandStringRunes(64)
	}

	transitions := make([]*types.BalanceTransition, 0, NT)
	for i := 0; i < NT; i++ {
		transitions = append(transitions, &types.BalanceTransition{
			Path:   addresses[rand.Intn(NA)],
			Tx:     uint32(i),
			Origin: new(big.Int).SetUint64(1000000000),
			Delta:  new(big.Int).SetInt64(-rand.Int63n(100)),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accumulator.Check(transitions, NT)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
