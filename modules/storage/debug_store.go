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
	"context"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type LogSaveRequest struct {
	Height   uint64
	Execlogs []string
}
type StatisticInfoSaveRequest struct {
	Height          uint64
	StatisticalInfo *mtypes.StatisticalInformation
}

type DebugStore struct {
	// execlog  *mstypes.ExeclogCaches
	exectime *mstypes.ExectimeCaches
}

func NewDebugStore() *DebugStore {
	return &DebugStore{
		// TODO
		// execlog:  NewExeclogCaches(50, 8),
		// exectime: NewExectimeCaches(50),
	}
}

func (ds *DebugStore) Config(params map[string]interface{}) {
	ds.exectime = mstypes.NewExectimeCaches(int(params["cache_statistcalinfo_size"].(float64)))
	// ds.execlog = mstypes.NewExeclogCaches(int(params["cache_execlog_size"].(float64)), int(params["cache_exec_concurrency"].(float64)))
}

func (ds *DebugStore) SaveLog(ctx context.Context, request *LogSaveRequest, _ *int) error {
	// ds.execlog.Save(request.Height, request.Execlogs)
	return nil
}

func (ds *DebugStore) GetExecLog(ctx context.Context, hash *evmCommon.Hash, log *string) error {
	// log = ds.execlog.Query(string(hash.Bytes()))
	return nil
}

func (ds *DebugStore) SaveStatisticInfos(ctx context.Context, request *StatisticInfoSaveRequest, _ *int) error {
	ds.exectime.Save(request.Height, request.StatisticalInfo)
	return nil
}

func (ds *DebugStore) GetStatisticInfos(ctx context.Context, height *uint64, staticalInfo **mtypes.StatisticalInformation) error {
	*staticalInfo = ds.exectime.Query(*height)
	return nil
}
