package types

import (
	"fmt"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

const (
	SlicePerSyncPoint = 256
)

type SyncStatus struct {
	Id        string
	SyncPoint uint64
	Height    uint64

	// FIXME
	RequestId string
}

type SyncPoint struct {
	From       uint64
	To         uint64
	Slices     []ethCommon.Hash
	Parent     *ParentInfo
	SchdStates interface{}
}

type SyncDataRequest struct {
	From  uint64
	To    uint64
	Slice int

	// FIXME
	StartAt time.Time
}

type SyncDataResponse struct {
	SyncDataRequest
	// TBD
	Hash       []byte
	Data       []byte
	Parent     *ParentInfo
	SchdStates interface{}
}

func (sdr *SyncDataRequest) ID() string {
	return fmt.Sprintf("%d|%d", sdr.To, sdr.Slice)
}
