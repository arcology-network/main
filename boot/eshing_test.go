//go:build !CI

package boot

import (
	"testing"
	"time"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/mock/kafka"
	"github.com/arcology-network/component-lib/mock/rpc"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/main/config"
)

func TestEshingSvc(t *testing.T) {

	config.DownloaderCreator = kafka.NewDownloaderCreator(t)
	config.UploaderCreator = kafka.NewUploaderCreator(t)
	intf.RPCCreator = rpc.NewRPCServerInitializer(t)

	globalConfig := config.LoadGlobalConfig("../config/global.json")
	kafkaConfig := config.LoadKafkaConfig("../config/kafka.json")
	appConfig := config.LoadAppConfig("../modules/storage/storage.json")
	brk, _, uploaders := initApp(globalConfig, kafkaConfig, appConfig)

	broker := &actor.MessageWrapper{
		MsgBroker:      brk,
		LatestMessage:  actor.NewMessage(),
		WorkThreadName: "unittester",
	}

	broker.Send(actor.MsgBlockCompleted, actor.MsgBlockCompleted_Success)

	h1 := evmCommon.BytesToHash([]byte("h1"))
	h2 := evmCommon.BytesToHash([]byte("h2"))
	broker.Send(actor.MsgEuResults, &types.Euresults{{H: string(h1.Bytes())}}, 1)
	broker.Send(actor.MsgEuResults, &types.Euresults{{H: string(h2.Bytes())}}, 1)
	broker.Send(actor.MsgInclusive, &types.InclusiveList{
		HashList:   []*evmCommon.Hash{&h1, &h2},
		Successful: []bool{true, true},
	}, 1)

	time.Sleep(time.Second * 3)
	t.Log(uploaders[0].(*kafka.Uploader).GetCounter())

}
