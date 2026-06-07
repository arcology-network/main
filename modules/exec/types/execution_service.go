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
	"github.com/arcology-network/eu/eu"
	eushared "github.com/arcology-network/eu/shared"
	workload "github.com/arcology-network/scheduler/workload"
	statecache "github.com/arcology-network/state-engine/state/cache"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ExecMessagers struct {
	Snapshot *statecache.ExecutionStateStore
	Config   *eu.Config
	Sequence *workload.JobSequence
}

type ExecutionResponse struct {
	AccessRecords   []*eushared.TxAccessRecords
	EuResults       []*eushared.EuResult
	Receipts        []*evmTypes.Receipt
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
	ExecutingLogs   []string
}
