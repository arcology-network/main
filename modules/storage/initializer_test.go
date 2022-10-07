package storage

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"

	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	cmntyp "github.com/HPISTechnologies/common-lib/types"
)

func TestLoadFile(t *testing.T) {
	filename := "/home/weizp/work/af"
	loadGenesisAccounts(filename)
}
func loadGenesisAccounts(filename string) map[ethcmn.Address]*cmntyp.Account {
	file, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	accounts := make(map[ethcmn.Address]*cmntyp.Account)
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

		accounts[ethcmn.HexToAddress(segments[1])] = &cmntyp.Account{
			Nonce:   1,
			Balance: balance,
		}
	}
	return accounts
}
