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

package storage

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const (
	AccountNum  = 5000000
	ContractNum = 1000
)

func TestMetaIndexer(t *testing.T) {
	begin := time.Now()
	keys := make([]string, 0, AccountNum*10)
	values := make([][]byte, 0, AccountNum*10)

	for i := 0; i < AccountNum; i++ {
		keys = append(keys, generateAccountUrls()...)
	}
	for i := 0; i < len(keys); i++ {
		values = append(values, []byte{0})
	}
	t.Log("time to generate test data:", time.Since(begin))

	begin = time.Now()
	indexer := NewMetaIndexer()
	indexer.Scan(keys, values)
	indexer.Stop()
	t.Log("time to build index:", time.Since(begin))

	indexer.PrintSummary()
}

func BenchmarkMetaIndexer(b *testing.B) {
	begin := time.Now()
	keys := make([]string, 0, ContractNum*60000)
	values := make([][]byte, 0, ContractNum*60000)

	for i := 0; i < AccountNum; i++ {
		keys = append(keys, generateAccountUrls()...)
	}
	for i := 0; i < ContractNum; i++ {
		keys = append(keys, generateContractUrls()...)
	}
	for i := 0; i < len(keys); i++ {
		values = append(values, []byte{0})
	}
	b.Log("time to generate test data:", time.Since(begin))

	begin = time.Now()
	indexer := NewMetaIndexer()
	indexer.Scan(keys, values)
	indexer.Stop()
	b.Log("time to build index:", time.Since(begin))

	begin = time.Now()
	ks, _ := indexer.ExportMetas()
	b.Log("time to export metas:", time.Since(begin), ", num of metas:", len(ks))
	indexer.PrintSummary()
}

func generateAccountUrls() []string {
	address := randomHexString(20)
	return []string{
		RootPrefix + address + "/code",
		RootPrefix + address + "/nonce",
		RootPrefix + address + "/balance",
		RootPrefix + address + "/defer",
	}
}

func generateContractUrls() []string {
	address := randomHexString(20)
	keys := make([]string, 0, 60000)
	// Basic account keys.
	keys = append(keys, []string{
		RootPrefix + address + "/code",
		RootPrefix + address + "/nonce",
		RootPrefix + address + "/balance",
		RootPrefix + address + "/defer",
	}...)
	// Containers.
	for i := 0; i < 5; i++ {
		container := randomHexString(5)
		// Basic container keys.
		keys = append(keys, []string{
			RootPrefix + address + "/storage/containers/" + container + "/",
			RootPrefix + address + "/storage/containers/" + container + "/@",
			RootPrefix + address + "/storage/containers/" + container + "/!",
			RootPrefix + address + "/storage/containers/!/" + container,
		}...)
		// Elements in a container.
		for j := 0; j < 10000; j++ {
			element := randomHexString(20)
			keys = append(keys, RootPrefix+address+"/storage/containers/"+container+"/"+element)
		}
	}
	// Native storage.
	for i := 0; i < 100; i++ {
		element := randomHexString(32)
		keys = append(keys, RootPrefix+address+"/storage/native/"+element)
	}

	return keys
}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomHexString(nbytes int) string {
	b := make([]byte, nbytes)
	rnd.Read(b)
	return fmt.Sprintf("%x", b)
}
