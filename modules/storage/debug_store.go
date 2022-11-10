package storage

import (
	"context"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	"github.com/arcology-network/common-lib/types"
	mstypes "github.com/arcology-network/main/modules/storage/types"
)

type LogSaveRequest struct {
	Height   uint64
	Execlogs []string
}
type StatisticInfoSaveRequest struct {
	Height          uint64
	StatisticalInfo *types.StatisticalInformation
}

type DebugStore struct {
	execlog  *mstypes.ExeclogCaches
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
	ds.execlog = mstypes.NewExeclogCaches(int(params["cache_execlog_size"].(float64)), int(params["cache_exec_concurrency"].(float64)))
}

func (ds *DebugStore) SaveLog(ctx context.Context, request *LogSaveRequest, _ *int) error {
	ds.execlog.Save(request.Height, request.Execlogs)
	return nil
}

func (ds *DebugStore) GetExecLog(ctx context.Context, hash *ethcmn.Hash, log *string) error {
	log = ds.execlog.Query(string(hash.Bytes()))
	return nil
}

func (ds *DebugStore) SaveStatisticInfos(ctx context.Context, request *StatisticInfoSaveRequest, _ *int) error {
	ds.exectime.Save(request.Height, request.StatisticalInfo)
	return nil
}

func (ds *DebugStore) GetStatisticInfos(ctx context.Context, height *uint64, staticalInfo **types.StatisticalInformation) error {
	*staticalInfo = ds.exectime.Query(*height)
	return nil
}
