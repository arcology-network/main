package types

import "encoding/gob"

func init() {
	gob.Register(&RPCTransaction{})
	gob.Register(&RPCBlock{})
}
