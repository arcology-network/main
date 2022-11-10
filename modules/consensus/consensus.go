package consensus

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"path"
	"sync"
	"time"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	tmCommon "github.com/arcology-network/3rd-party/tm/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/consensus-engine/cmd/tendermint/commands"
	"github.com/arcology-network/consensus-engine/config"
	tmlog "github.com/arcology-network/consensus-engine/libs/log"
	"github.com/arcology-network/consensus-engine/monaco"
	"github.com/arcology-network/consensus-engine/node"
	"github.com/arcology-network/consensus-engine/p2p"
	"github.com/arcology-network/consensus-engine/privval"
	"github.com/arcology-network/consensus-engine/proxy"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"
	"go.uber.org/zap"
)

type Consensus struct {
	actor.WorkerThread
	pendingMsgs     map[string]chan *actor.Message
	maxTxsNum       int
	rate            int64
	starter         int64
	storageSvcName  string
	debug           bool
	cachedMetaBlock *actor.Message
	GotBlock        bool
	chanTxs         chan [][]byte
	isproposer      bool
	syncing         bool
	engineCfg       *config.Config
}

var (
	consensusSingleton actor.IWorkerEx
	initOnce           sync.Once
)

// return a Subscriber struct
func NewConsensus(concurrency int, groupid string) actor.IWorkerEx {
	initOnce.Do(func() {
		c := Consensus{}
		c.Set(concurrency, groupid)
		c.pendingMsgs = map[string]chan *actor.Message{
			actor.MsgAppHash:   make(chan *actor.Message, 10),
			actor.MsgMetaBlock: make(chan *actor.Message, 10),
		}
		c.GotBlock = false
		c.chanTxs = make(chan [][]byte, 10000)
		c.isproposer = true
		c.syncing = false

		consensusSingleton = &c
	})

	return consensusSingleton
}

func (c *Consensus) Inputs() ([]string, bool) {
	return []string{actor.MsgExtAppHash, actor.MsgMetaBlock, actor.MsgTxLocals}, false
}

func (c *Consensus) Outputs() map[string]int {
	return map[string]int{
		actor.MsgExtReapCommand:         1,
		actor.MsgExtTxBlocks:            1,
		actor.MsgExtBlockCompleted:      1,
		actor.MsgExtReapingList:         1,
		actor.MsgExtBlockStart:          1,
		actor.MsgConsensusMaxPeerHeight: 1,
		actor.MsgConsensusUp:            1,
		actor.MsgBlockEnd:               1,
	}
}

func (c *Consensus) Config(params map[string]interface{}) {
	c.maxTxsNum = int(params["max_tx_num"].(float64))
	c.rate = int64(params["rate"].(float64))
	c.starter = int64(params["starter"].(float64))
	c.storageSvcName = params["storage_svc_name"].(string)
	intf.Router.SetZkServers([]string{params["zookeeper"].(string)})
	c.debug = params["debug"].(bool)

	cfg, err := commands.ParseConfig()
	if err != nil {
		panic(err)
	}
	c.engineCfg = cfg
	c.engineCfg.P2P.PersistentPeers = params["persistent_peers"].(string)
	c.engineCfg.Instrumentation.Prometheus = true
	c.engineCfg.Instrumentation.PrometheusListenAddr = params["prometheus_listen_addr"].(string)
	c.engineCfg.P2P.SendRate = 5120000 * 20 //100m
	c.engineCfg.P2P.RecvRate = 5120000 * 20 //100m
}

func (c *Consensus) OnStart() {

}

func (c *Consensus) InitMsgs() []*actor.Message {
	err := c.startConsensus(c, c.engineCfg)
	if err != nil {
		panic(err)
	}
	return nil
}

func (c *Consensus) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgMetaBlock:
			c.pendingMsgs[actor.MsgMetaBlock] <- v
		case actor.MsgExtAppHash:
			c.pendingMsgs[actor.MsgAppHash] <- v
		case actor.MsgTxLocals:
			c.chanTxs <- v.Data.([][]byte)
		}
	}
	return nil
}

func (c *Consensus) Query(ctx context.Context, request *types.QueryRequest, response *types.QueryResult) error {
	switch request.QueryType {
	case types.QueryType_Syncing:
		response.Data = c.syncing
	case types.QueryType_Proposer:
		response.Data = c.isproposer
	}
	return nil
}

func (c *Consensus) Proposer(isporposer bool) {
	c.isproposer = isporposer
}
func (c *Consensus) Syncing(syncing bool) {
	c.syncing = syncing
}
func (c *Consensus) Reap(maxBytes int64, maxGas int64) (txs [][]byte, hashes [][]byte) {
	c.AddLog(log.LogLevel_Debug, "enter Reap", zap.Uint64("height", c.LatestMessage.Height))
	if c.cachedMetaBlock != nil {
		c.AddLog(log.LogLevel_Debug, "c.cachedMetaBlock.Height", zap.Uint64("cache height", c.cachedMetaBlock.Height))
	}
	var msg *actor.Message
	if c.cachedMetaBlock != nil && c.LatestMessage.Height == c.cachedMetaBlock.Height {
		msg = c.cachedMetaBlock
		c.AddLog(log.LogLevel_Debug, "got cachedMetaBlock")
	} else {
		c.AddLog(log.LogLevel_Debug, "waiting MetaBlock")

		msg = <-c.pendingMsgs[actor.MsgMetaBlock]
		c.AddLog(log.LogLevel_Debug, "got MetaBlock")
		c.GotBlock = true
	}
	c.cachedMetaBlock = msg
	c.AddLog(log.LogLevel_Debug, "after c.cachedMetaBlock.Height", zap.Uint64("cache height", c.cachedMetaBlock.Height))

	metaBlock := msg.Data.(*types.MetaBlock)
	txs = [][]byte{}
	hashes = [][]byte{}
	if metaBlock != nil {
		txs = metaBlock.Txs
		hashes = make([][]byte, len(metaBlock.Hashlist))
		for i, h := range metaBlock.Hashlist {
			hashes[i] = h.Bytes()
		}
	}
	c.AddLog(log.LogLevel_Debug, "return Reap", zap.Int("hashes", len(hashes)))
	return
}
func (c *Consensus) AddToMempool(txs [][]byte, src string) {
	c.AddLog(log.LogLevel_Info, "AddToMempool", zap.Int("txs", len(txs)))
	groups := c.parseGroups(txs)
	for i := range groups {
		if len(groups[i]) > 0 {
			c.MsgBroker.Send(actor.MsgExtTxBlocks, &types.IncomingTxs{
				Txs: groups[i],
				Src: types.NewTxSource(types.TxSourceConsensus, src),
			})
		}
	}
}

func (c *Consensus) parseGroups(txs [][]byte) [][][]byte {
	txLen := len(txs)
	idx := 0
	groups := [][][]byte{}
	for {
		beginindex := idx * c.maxTxsNum
		endindex := int(math.Min(float64((idx+1)*c.maxTxsNum), float64(txLen)))

		list := txs[beginindex:endindex]
		data := make([][]byte, len(list))
		for k := range list {
			data[k] = []byte(list[k])
		}
		groups = append(groups, data)

		idx = idx + 1
		if endindex == txLen {
			break
		}
	}
	return groups
}
func (c *Consensus) ApplyTxsSync(height int64, coinbase []byte, timestamp time.Time, hashes [][]byte) []byte {
	latestMsg := actor.Message{
		Msgid:  log.Logger.GetLogId(),
		Name:   actor.MsgBlockStart,
		Height: uint64(height - 1),
		Round:  0,
	}

	c.ChangeEnvironment(&latestMsg)

	c.AddLog(log.LogLevel_Info, "ApplyTxsSync")

	if height > 1 {
		c.MsgBroker.Send(actor.MsgExtBlockCompleted, actor.MsgBlockCompleted_Success, uint64(height-1))
	}
	latestMsg.Height = uint64(height)
	c.ChangeEnvironment(&latestMsg)

	var na int
	txID := fmt.Sprintf("%d", height)
	intf.Router.Call("transactionalstore", "BeginTransaction", &txID, &na)

	reapHashlist := make([]*ethCommon.Hash, len(hashes))
	for i, h := range hashes {
		hash := ethCommon.BytesToHash(h)
		reapHashlist[i] = &hash
	}
	c.AddLog(log.LogLevel_Info, "start send reapinglist", zap.Int("reapinglist hashes length", len(reapHashlist)))
	c.MsgBroker.Send(actor.MsgExtReapingList, &types.ReapingList{
		List:      reapHashlist,
		Timestamp: big.NewInt(0),
	}, uint64(height))
	c.CheckPoint("send reapinglist")

	coinbaseAddress := ethCommon.BytesToAddress(coinbase)
	multiResult := big.NewInt(0).Mul(big.NewInt(timestamp.Unix()), big.NewInt(c.rate))
	blockstamp := big.NewInt(0).Add(big.NewInt(c.starter), multiResult)
	c.MsgBroker.Send(actor.MsgExtBlockStart, &actor.BlockStart{
		Timestamp: blockstamp,
		Coinbase:  coinbaseAddress,
		Height:    uint64(height),
	}, uint64(height))
	c.CheckPoint("block start")

	c.AddLog(log.LogLevel_Debug, "[ApplyTxsSync] Before got block.")
	if !c.GotBlock {
		<-c.pendingMsgs[actor.MsgMetaBlock]
	}
	c.AddLog(log.LogLevel_Debug, "[ApplyTxsSync] After got block.")
	c.GotBlock = false
	msg := <-c.pendingMsgs[actor.MsgAppHash]
	intf.Router.Call("transactionalstore", "EndTransaction", &na, &na)
	c.AddLog(log.LogLevel_Debug, "[ApplyTxsSync] After got apphash.")
	c.MsgBroker.Send(actor.MsgBlockEnd, "", uint64(height))
	c.MsgBroker.Send(actor.MsgExtReapCommand, "", uint64(height))
	return msg.Data.([]byte)
}

func (c *Consensus) GetLocalTxsChan() chan [][]byte {
	return c.chanTxs
}

func (c *Consensus) GetTxsOnBlock(height uint64) ([][]byte, error) {
	request := types.QueryRequest{
		QueryType: types.QueryType_RawBlock,
		Data:      height,
	}
	response := types.QueryResult{}
	err := intf.Router.Call(c.storageSvcName, "Query", &request, &response)
	if err != nil {
		return nil, err
	}

	return (*response.Data.(*types.MonacoBlock)).Txs, nil
}

func (c *Consensus) CreateBlockStore() monaco.BlockStore {
	return newBlockStore("tmblockstore")
}

func (c *Consensus) CreateStateStore() interface{} {
	return newStateStore("tmstatestore")
}

func (c *Consensus) UpdateMaxPeerHeight(height uint64) {
	c.MsgBroker.Send(actor.MsgConsensusMaxPeerHeight, height)
}

func (c *Consensus) SwitchToConsensus() {
	c.MsgBroker.Send(actor.MsgConsensusUp, "")
}

func (c *Consensus) startConsensus(backend monaco.BackendProxy, config *config.Config) error {
	logname := "consensus.log"
	rootDir := viper.GetString(cli.HomeFlag)
	//create logger
	if err := tmCommon.EnsureDir(path.Join(rootDir, "log"), 0777); err != nil {
		tmCommon.PanicSanity(err.Error())
	}
	logfile, err := os.OpenFile(path.Join(rootDir, "log", logname), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		tmCommon.PanicSanity(err.Error())
	}

	// logname := "consensus.log"
	// //create logger
	// if err := tmCommon.EnsureDir(path.Join(c.rootDir, "log"), 0777); err != nil {
	// 	tmCommon.PanicSanity(err.Error())
	// }
	// logfile, err := os.OpenFile(path.Join(c.rootDir, "log", logname), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	// if err != nil {
	// 	tmCommon.PanicSanity(err.Error())
	// }

	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(logfile))
	if c.debug {
		logger = tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	}
	//return logger
	logger = logger.With("svc", "consensus")
	finename := config.NodeKeyFile()
	nodeKey, err := p2p.LoadOrGenNodeKey(finename)
	if err != nil {
		return fmt.Errorf("failed to load or gen node key %s: %w", config.NodeKeyFile(), err)
	}

	n, err := node.NewNodeEx(config,
		privval.LoadOrGenFilePVEx(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(&FakeApp{}),
		node.DefaultGenesisDocProviderFunc(config),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(config.Instrumentation),
		logger,
		backend,
	)
	if err != nil {
		fmt.Printf("err=%v\n", err)
		return err
	}

	err = n.Start()
	if err != nil {
		return err
	}

	return nil
}
