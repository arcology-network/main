/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package types

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/arcology-network/common-lib/codec"
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
	filehandle := NewRawFiles("testdata")

	data := make([][]byte, 50000)
	for i := range data {
		data[i] = make([]byte, 1000)
	}

	val := codec.Byteset(data).Encode()
	t0 := time.Now()
	filehandle.Write(filehandle.GetFilename(345), val)
	fmt.Printf("==============write time : %v\n", time.Since(t0))

	t1 := time.Now()
	filedata, err := filehandle.Read(filehandle.GetFilename(345))
	fmt.Printf("==============read time : %v\n", time.Since(t1))
	if err != nil {
		t.Error("raw file read err:" + err.Error())
	}

	if !reflect.DeepEqual(val, filedata) {
		t.Error("raw file read write err")
	}
	fmt.Printf("==============write time 2: %v\n", time.Since(t0))
}
