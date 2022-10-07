package types

import (
	"fmt"
	"testing"
)

func TestRead(t *testing.T) {
	ll, _ := LoadingConf("conflictlist")
	for i, item := range ll {
		fmt.Printf("left ---------------idx=%v\n", i)
		for _, entrance := range item.Left {
			fmt.Printf("left ---------------  address=%v\n", entrance.ContractAddress)
		}
		fmt.Printf("right ---------------\n")
		for _, entrance := range item.Right {
			fmt.Printf("left ---------------  address=%v\n", entrance.ContractAddress)
		}
	}

}
