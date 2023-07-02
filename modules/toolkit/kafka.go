package toolkit

import (
	"fmt"
	"strings"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	commutative "github.com/arcology-network/concurrenturl/commutative"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
)

type kafka struct {
	actor.WorkerThread
	queryHeight uint64
	queryKey    string
}

// return a Subscriber struct
func Newkafka(concurrency int, groupid string) actor.IWorkerEx {
	ka := kafka{}
	ka.Set(concurrency, groupid)
	return &ka
}
func (c *kafka) Config(params map[string]interface{}) {
	c.queryHeight = uint64(params["queryheight"].(float64))
	c.queryKey = params["querykey"].(string)
}
func (c *kafka) Inputs() ([]string, bool) {
	return []string{actor.MsgEuResults}, false
}

func (c *kafka) Outputs() map[string]int {
	return map[string]int{}
}

func (c *kafka) OnStart() {
}

func (c *kafka) OnMessageArrived(msgs []*actor.Message) error {
	total := 0
	cc := 0
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgEuResults:
			if v.Height == c.queryHeight {
				data := v.Data.(*types.Euresults)
				if data != nil {
					for i := range *data {
						transitions := (*data)[i].Transitions
						transitionData := univaluepk.UnivaluesDecode(transitions, func() interface{} { return &univaluepk.Univalue{} }, nil)
						size := 0

						for j := range transitionData {
							key := *transitionData[j].GetPath()
							if strings.Contains(key, c.queryKey) {
								size = size + 1
								total = total + 1
								nonce := transitionData[j].Value().(*commutative.Uint64).Delta().(uint64)

								if nonce > 1 {
									fmt.Printf("=====height=%v======h=%x   %v\n", v.Height, []byte((*data)[i].H), nonce)
								} else if nonce == 1 {
									cc = cc + 1
								}

							}
						}
						if size > 1 {
							fmt.Printf("=====height=%v======h=%x   %v\n", v.Height, []byte((*data)[i].H), size)
						}
					}
				}
			}
			fmt.Printf("======height=%v  total:%v cc:%v\n", v.Height, total, cc)
		}
	}

	return nil
}
