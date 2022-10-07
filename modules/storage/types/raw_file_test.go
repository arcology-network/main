package types

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/HPISTechnologies/common-lib/codec"
)

func TestOp(t *testing.T) {
	filenum1 := uint64(10)
	height := uint64(312)
	level1 := height / filenum1 / filenum1
	level2 := height / filenum1 % filenum1
	leaf := height % filenum
	fmt.Printf("%v/%v/%v\n", level1, level2, leaf)

	dirs := []string{"111", "222", "333"}
	for _, dir := range dirs[:len(dirs)-1] {
		fmt.Printf("=====dir=%v\n", dir)
	}
}
func TestRawFile(t *testing.T) {
	filehandle := NewRawFiles("datas")

	datas := make([][]byte, 50000)
	for i := range datas {
		datas[i] = make([]byte, 1000)
	}

	val := codec.Byteset(datas).Encode()
	t0 := time.Now()
	filehandle.Write(filehandle.GetFilename(345), val)
	fmt.Printf("==============write time : %v\n", time.Since(t0))

	t1 := time.Now()
	data, err := filehandle.Read(filehandle.GetFilename(345))
	fmt.Printf("==============read time : %v\n", time.Since(t1))
	if err != nil {
		t.Error("raw file read err:" + err.Error())
	}

	if !reflect.DeepEqual(val, data) {
		t.Error("raw file read write err")
	}
	fmt.Printf("==============write time 2: %v\n", time.Since(t0))
}
