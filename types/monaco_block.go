package types

import codec "github.com/arcology-network/common-lib/codec"

// "github.com/arcology-network/common-lib/common"

const (
	AppType_Eth  = 0
	AppType_Coin = 1
)

type MonacoBlock struct {
	Height    uint64
	Blockhash []byte
	Headers   [][]byte
	Txs       [][]byte
	Signer    uint8
}

func (mb MonacoBlock) Hash() []byte {
	// bys := [][]byte{codec.Byteset(mb.Headers).Flatten(), codec.Byteset(mb.Txs).Flatten(), common.Uint64ToBytes(mb.Height)}
	// sum := sha256.Sum256(codec.Byteset(bys).Flatten())
	// return sum[:]
	return mb.Blockhash
}

func (mb MonacoBlock) GobEncode() ([]byte, error) {
	data := [][]byte{
		codec.Uint64(mb.Height).Encode(),
		codec.Byteset(mb.Headers).Encode(),
		codec.Byteset(mb.Txs).Encode(),
		mb.Blockhash,
		[]byte{mb.Signer},
	}
	return codec.Byteset(data).Encode(), nil
}
func (mb *MonacoBlock) GobDecode(data []byte) error {
	fields := codec.Byteset{}.Decode(data).(codec.Byteset)
	mb.Height = uint64(codec.Uint64(0).Decode(fields[0]).(codec.Uint64))
	mb.Headers = codec.Byteset{}.Decode(fields[1]).(codec.Byteset)
	mb.Txs = codec.Byteset{}.Decode(fields[2]).(codec.Byteset)
	mb.Blockhash = fields[3]
	mb.Signer = uint8(fields[4][0])
	return nil
}
