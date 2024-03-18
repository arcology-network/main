package types

import (
	"fmt"
	"testing"
)

func TestRead(t *testing.T) {
	ll, _ := LoadingConf("conflictlist")
	for _, item := range ll {
		fmt.Printf("leftAddr:%v,leftSign:%v,rightAddr:%v,rightSign:%v\n", item.LeftAddr, item.LeftSign, item.RightAddr, item.RightSign)
	}

}
