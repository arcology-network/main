//go:build !CI

package storage

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

func TestLoadFile(t *testing.T) {
	filename := "../../config/af"
	loadGenesisAccounts(filename)
}

func loadGenesisAccounts(filename string) map[evmCommon.Address]*types.Account {
	file, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	accounts := make(map[evmCommon.Address]*types.Account)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}

		line = strings.TrimRight(line, "\n")
		segments := strings.Split(line, ",")
		balance, ok := new(big.Int).SetString(segments[2], 10)
		if !ok {
			panic(fmt.Sprintf("invalid balance in genesis accounts: %v", segments[2]))
		}

		accounts[evmCommon.HexToAddress(segments[1])] = &types.Account{
			Nonce:   0,
			Balance: balance,
		}
	}
	return accounts
}
