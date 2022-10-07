package arbitrator_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/HPISTechnologies/common-lib/common"
)

func TestStringAppend(t *testing.T) {
	begin := time.Now()
	var str string
	for i := 0; i < 50000; i++ {
		str += "blcc://eth1.0/accounts/0000111122223333444455556666777788889999/storage/containers/balance/$0000111122223333444455556666777788889999"
	}
	t.Log(len(str))
	t.Log(time.Duration(time.Since(begin)))
}

func TestStringAppend2(t *testing.T) {
	begin := time.Now()
	var buf bytes.Buffer
	for i := 0; i < 50000; i++ {
		fmt.Fprintf(&buf, "%s", "blcc://eth1.0/accounts/0000111122223333444455556666777788889999/storage/containers/balance/$0000111122223333444455556666777788889999")
	}
	str := buf.String()
	t.Log(len(str))
	t.Log(time.Duration(time.Since(begin)))
}

func TestJson(t *testing.T) {
	txids := []uint32{10, 12}
	paths := []string{"123", "456"}
	pathdata, err := json.Marshal(paths)
	if err != nil {
		fmt.Printf("Marshal err : %v\n", err)
		return
	}
	fmt.Printf("pathdata:%v\n", string(pathdata))

	common.AppendToFile("testapp", string(pathdata))

	txdata, err := json.Marshal(txids)
	if err != nil {
		fmt.Printf("Marshal err : %v\n", err)
		return
	}
	fmt.Printf("txdata:%v\n", string(txdata))

	common.AppendToFile("testapp", string(txdata))

	filename := "testapp"
	common.AddToLogFile(filename, "path", paths)
	common.AddToLogFile(filename, "txids", txids)
}
