package types

import (
	urlcommon "github.com/HPISTechnologies/concurrenturl/v2/common"
)

type ApcBlock struct {
	DB        *urlcommon.DatastoreInterface
	ApcHeight uint64
	Result    string
}
