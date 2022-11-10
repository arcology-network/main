package pool

type TxSourceStatistics struct {
	NumValid    uint64
	NumInvalid  uint64
	NumDup      uint64
	NumLowNonce uint64
}

func NewTxSourceStatistics() *TxSourceStatistics {
	return &TxSourceStatistics{}
}
