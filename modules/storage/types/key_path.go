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
	"strings"

	adaptorcommon "github.com/arcology-network/evm-adaptor/pathbuilder"
	evmcommon "github.com/ethereum/go-ethereum/common"
)

const (
	nthread = 4
)

var connector *adaptorcommon.EthPathBuilder

func init() {
	connector = &adaptorcommon.EthPathBuilder{}
}

func getBalancePath(addr string) string {
	return connector.BalancePath(evmcommon.HexToAddress(addr))
}
func getNoncePath(addr string) string {
	return connector.NoncePath(evmcommon.HexToAddress(addr))
}
func getCodePath(addr string) string {
	return connector.CodePath(evmcommon.HexToAddress(addr))
}

func getStorageKeyPath(addr, key string) string {
	if !strings.HasPrefix(key, "0x") {
		key = "0x" + key
	}

	return connector.StorageRootPath(evmcommon.HexToAddress(addr)) + key
}
