package protocol

import "encoding/gob"

func init() {
	gob.Register(&Message{})
}
