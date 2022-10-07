package gateway

import (
	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	gatewayTypes "github.com/HPISTechnologies/main/modules/gateway/types"
)

type RpcClient struct {
	actor.WorkerThread
}

//return a Subscriber struct
func NewRpcClient(concurrency int, groupid string) actor.IWorkerEx {
	rc := RpcClient{}
	rc.Set(concurrency, groupid)
	return &rc
}

func (rc *RpcClient) Inputs() ([]string, bool) {
	return []string{actor.MsgTxLocalsRpc}, false
}

func (rc *RpcClient) Outputs() map[string]int {
	return map[string]int{}
}

func (rc *RpcClient) OnStart() {
}

func (rc *RpcClient) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgTxLocalsRpc:
			pack := v.Data.(*gatewayTypes.TxsPack)
			response := types.RawTransactionReply{}
			if len(pack.Txs) > 0 {
				intf.Router.Call("tpp", "ReceivedTransactionFromRpc", &types.RawTransactionArgs{Txs: pack.Txs[0]}, &response)
				pack.TxHashChan <- *response.TxHash.(*ethCommon.Hash)
			} else {
				pack.TxHashChan <- ethCommon.Hash{}
			}

		}
	}
	return nil
}
