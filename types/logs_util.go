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
	eth "github.com/ethereum/go-ethereum"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethtyp "github.com/ethereum/go-ethereum/core/types"
)

type LogCache struct {
	Logs      []*ethtyp.Log
	Height    uint64
	BlockHash ethcmn.Hash
}

func containAddress(addrs []ethcmn.Address, addr ethcmn.Address) bool {
	for _, ad := range addrs {
		if ad == addr {
			return true
		}
	}
	return false
}
func FiltereTopic(tpoicsFilter [][]ethcmn.Hash, tpoics []ethcmn.Hash) bool {
	if len(tpoicsFilter) > len(tpoics) {
		return false
	}
	for i := range tpoicsFilter {
		if len(tpoicsFilter[i]) == 0 {
			continue
		}
		found := false
		for j := range tpoicsFilter[i] {
			if tpoicsFilter[i][j] == tpoics[i] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func FilteLogs(logs []*ethtyp.Log, filter eth.FilterQuery) []*ethtyp.Log {
	fielteredLogs := make([]*ethtyp.Log, 0, len(logs))
	if filter.Addresses == nil || len(filter.Addresses) == 0 {
		fielteredLogs = logs
	} else {
		for _, log := range logs {
			if containAddress(filter.Addresses, log.Address) {
				fielteredLogs = append(fielteredLogs, log)
			}
		}
	}
	topicFielteredLogs := make([]*ethtyp.Log, 0, len(fielteredLogs))
	if filter.Topics == nil || len(filter.Topics) == 0 {
		topicFielteredLogs = fielteredLogs
	} else {
		for _, log := range fielteredLogs {
			if FiltereTopic(filter.Topics, log.Topics) {
				topicFielteredLogs = append(topicFielteredLogs, log)
			}
		}
	}
	return topicFielteredLogs
}
func ToLogs(receipts []*ethtyp.Receipt) []*ethtyp.Log {
	logsSize := 0
	for i := range receipts {
		logsSize += len(receipts[i].Logs)
	}
	logs := make([]*ethtyp.Log, 0, logsSize)

	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	return logs
}
