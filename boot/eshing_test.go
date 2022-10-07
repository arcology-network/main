package boot

import (
	"testing"
	"time"

	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	cmntyp "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/mock/kafka"
	"github.com/HPISTechnologies/component-lib/mock/rpc"
	"github.com/HPISTechnologies/main/config"
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

	h1 := ethcmn.BytesToHash([]byte("h1"))
	h2 := ethcmn.BytesToHash([]byte("h2"))
	broker.Send(actor.MsgEuResults, &cmntyp.Euresults{{H: string(h1.Bytes())}}, 1)
	broker.Send(actor.MsgEuResults, &cmntyp.Euresults{{H: string(h2.Bytes())}}, 1)
	broker.Send(actor.MsgInclusive, &cmntyp.InclusiveList{
		HashList:   []*ethcmn.Hash{&h1, &h2},
		Successful: []bool{true, true},
	}, 1)

	time.Sleep(time.Second * 3)
	t.Log(uploaders[0].(*kafka.Uploader).GetCounter())
}
