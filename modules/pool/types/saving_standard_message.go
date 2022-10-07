package types

import (
	"github.com/HPISTechnologies/common-lib/types"
)

type SavingStandardMessage struct {
	Msg     *types.StandardMessage
	RawData []byte
}
