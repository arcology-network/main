package types

import (
	"github.com/arcology-network/common-lib/types"
)

type SavingStandardMessage struct {
	Msg     *types.StandardMessage
	RawData []byte
}
