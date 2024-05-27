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
	"strings"
	"sync"

	"github.com/arcology-network/common-lib/common"
	concurrenturlcommon "github.com/arcology-network/storage-committer/common"
	"github.com/arcology-network/storage-committer/commutative"
)

var (
	RootPrefix         = concurrenturlcommon.ETH10_ACCOUNT_PREFIX //ccurl.NewPlatform().Eth10Account()
	ContainerPrefix    = "/storage/containers/"
	RootPrefixLen      = len(RootPrefix)
	AddressPrefixLen   = len(RootPrefix) + 40
	ContainerPrefixLen = len(ContainerPrefix)
	TotalPrefixLen     = AddressPrefixLen + ContainerPrefixLen
)

const (
	SliceNum = 16
)

type update struct {
	key      string
	isDelete bool
}

type MetaIndexer struct {
	// [slice]map[container][element]{}
	indices [SliceNum]map[string]map[string]struct{}
	chs     [SliceNum]chan *update
	wg      sync.WaitGroup
}

func NewMetaIndexer() *MetaIndexer {
	indexer := &MetaIndexer{}
	for i := 0; i < SliceNum; i++ {
		indexer.indices[i] = make(map[string]map[string]struct{})
		indexer.chs[i] = make(chan *update, 100)

		indexer.wg.Add(1)
		go func(slice int) {
			for update := range indexer.chs[slice] {
				if update == nil {
					indexer.wg.Done()
					return
				}

				sep := strings.LastIndex(update.key, "/")
				container := update.key[:sep+1]
				element := update.key[sep+1:]
				// Container's root path.
				if len(element) == 0 {
					continue
				}

				if _, ok := indexer.indices[slice][container]; !ok {
					indexer.indices[slice][container] = make(map[string]struct{})
				}

				if update.isDelete {
					delete(indexer.indices[slice][container], element)
				} else {
					indexer.indices[slice][container][element] = struct{}{}
				}
			}
		}(i)
	}
	return indexer
}

func (indexer *MetaIndexer) Scan(keys []string, values [][]byte) {
	common.ParallelWorker(len(keys), 8, func(start, end, index int, args ...interface{}) {
		for i := start; i < end; i++ {
			if len(keys[i]) <= TotalPrefixLen {
				continue
			}

			if keys[i][TotalPrefixLen] == '!' || keys[i][AddressPrefixLen:TotalPrefixLen] != ContainerPrefix {
				continue
			}

			indexer.chs[hex2int(keys[i][RootPrefixLen])] <- &update{
				key:      keys[i],
				isDelete: len(values[i]) == 0,
			}
		}
	})
}

func (indexer *MetaIndexer) Stop() {
	for i := 0; i < SliceNum; i++ {
		indexer.chs[i] <- nil
	}
	indexer.wg.Wait()
}

func (indexer *MetaIndexer) PrintSummary() {
	totalContainerNum := 0
	totalElementNum := 0
	for i := 0; i < SliceNum; i++ {
		elemNum := 0
		for _, elems := range indexer.indices[i] {
			elemNum += len(elems)
		}
		totalContainerNum += len(indexer.indices[i])
		totalElementNum += elemNum
		fmt.Printf("[MetaIndexer] slice[%d], %d containers, %d elements\n", i, len(indexer.indices[i]), elemNum)
	}
	fmt.Printf("[MetaIndexer] %d containers, %d elements in total\n", totalContainerNum, totalElementNum)
}

func (indexer *MetaIndexer) ExportMetas() (keys []string, values []interface{}) {
	for i := 0; i < SliceNum; i++ {
		for container, elements := range indexer.indices[i] {

			// meta, _ := commutative.NewMeta(container)
			ks := make([]string, 0, len(elements))
			for e := range elements {
				ks = append(ks, e)
			}
			// meta.(*commutative.Meta).SetKeys(ks)

			path := commutative.NewPath()

			keys = append(keys, container)
			values = append(values, path)
		}
	}
	return
}

func hex2int(h byte) int {
	if h <= byte('9') {
		return int(h - byte('0'))
	} else {
		return int(h-byte('a')) + 10
	}
}
