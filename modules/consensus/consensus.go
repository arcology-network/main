/*
*   Copyright (c) 2024 Arcology Network

*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU General Public License as published by
*   the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.

*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU General Public License for more details.

*   You should have received a copy of the GNU General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
package consensus

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/consensus-engine/cmd/tendermint/commands"
	"github.com/arcology-network/consensus-engine/config"
	tmlog "github.com/arcology-network/consensus-engine/libs/log"
	tmos "github.com/arcology-network/consensus-engine/libs/os"
	"github.com/arcology-network/consensus-engine/monaco"
	"github.com/arcology-network/consensus-engine/node"
	"github.com/arcology-network/consensus-engine/p2p"
	"github.com/arcology-network/consensus-engine/privval"
	"github.com/arcology-network/consensus-engine/proxy"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"
)

type Consensus struct {
	pendingMsgs     map[string]chan *scommon.Message
	maxTxsNum       int
	rate            int64
	starter         int64
	storageSvcName  string
	debug           bool
	cachedMetaBlock *scommon.Message
	// GotBlock        bool
	chanTxs    chan [][]byte
	isproposer bool
	syncing    bool
	engineCfg  *config.Config
	isInited   bool
	sender     actor.OutboundSender
	height     uint64
	from       string
}

// return a Subscriber struct
func NewConsensus() actor.Business {

	c := Consensus{
		from: "consensus",
	}

	c.pendingMsgs = map[string]chan *scommon.Message{
		scommon.MsgExtAppHash: make(chan *scommon.Message, 10),
		scommon.MsgMetaBlock:  make(chan *scommon.Message, 10),
	}
	// c.GotBlock = false
	c.chanTxs = make(chan [][]byte, 10000)
	c.isproposer = true
	c.syncing = false

	return &c
}

func (c *Consensus) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgExtAppHash,
		scommon.MsgMetaBlock,
		scommon.MsgTxLocals,
		scommon.MsgInitialization,
	}, false
}

func (c *Consensus) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgExtReapCommand:         1,
		scommon.MsgExtTxBlocks:            1,
		scommon.MsgExtBlockCompleted:      1,
		scommon.MsgExtReapingList:         1,
		scommon.MsgExtBlockStart:          1,
		scommon.MsgConsensusMaxPeerHeight: 1,
		scommon.MsgConsensusUp:            1,
		scommon.MsgExtBlockEnd:            1,
	}
}

func (c *Consensus) Config(params map[string]interface{}) {
	c.maxTxsNum = int(params["max_tx_num"].(int))
	c.rate = int64(params["rate"].(int))
	c.starter = int64(params["starter"].(int))
	// c.storageSvcName = params["storage_svc_name"].(string)
	// intf.Router.SetZkServers([]string{params["zookeeper"].(string)})
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
	c.engineCfg.P2P.MaxNumInboundPeers = 100
	c.engineCfg.P2P.MaxNumOutboundPeers = 100
}

func (c *Consensus) RpcConfig() (string, int) {
	return "consensus", 20
}
func (c *Consensus) SetSender(sender actor.OutboundSender) {
	c.sender = sender
}
func (c *Consensus) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Query", c.Query)
	reg.Register(scommon.MsgMetaBlock, c.receivedMetaBlock)
	reg.Register(scommon.MsgExtAppHash, c.receivedExtAppHash)
	reg.Register(scommon.MsgTxLocals, c.receivedTxLocals)
	reg.Register(scommon.MsgInitialization, c.receivedInitialization)
}

func (c *Consensus) receivedMetaBlock(ctx *actor.ActionContext) error {
	c.pendingMsgs[scommon.MsgMetaBlock] <- ctx.Messages[0]
	return nil
}
func (c *Consensus) receivedExtAppHash(ctx *actor.ActionContext) error {
	c.pendingMsgs[scommon.MsgExtAppHash] <- ctx.Messages[0]
	return nil
}
func (c *Consensus) receivedTxLocals(ctx *actor.ActionContext) error {
	c.chanTxs <- ctx.Messages[0].Data.([][]byte)
	return nil
}
func (c *Consensus) receivedInitialization(ctx *actor.ActionContext) error {
	c.height = ctx.Messages[0].Data.(*mtypes.Initialization).BlockStart.Height
	err := c.startConsensus(c, c.engineCfg)
	if err != nil {
		panic(err)
	}
	return nil
}

func (c *Consensus) Query(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.QueryRequest)
	switch request.QueryType {
	case mtypes.QueryType_Syncing:
		ctx.ExecCtx.SendRpcResponse("", c.syncing)
	case mtypes.QueryType_Proposer:
		ctx.ExecCtx.SendRpcResponse("", c.isproposer)
	}
	return nil
}

func (c *Consensus) Proposer(isporposer bool) {
	c.isproposer = isporposer
}
func (c *Consensus) Syncing(syncing bool) {
	c.syncing = syncing
}
func (c *Consensus) Reap(maxBytes int64, maxGas int64, height int64) (txs [][]byte, hashes [][]byte) {
	c.height = uint64(height)
	logger.Log.Debug(context.Background(), c.from, "enter Reap", logger.F("height", height))

	var msg *scommon.Message
	if c.cachedMetaBlock != nil && uint64(height) == c.cachedMetaBlock.Height {
		msg = c.cachedMetaBlock
		logger.Log.Debug(context.Background(), c.from, "got from cachedMetaBlock")
	} else {
		logger.Log.Debug(context.Background(), c.from, "waiting MetaBlock")
		msg = <-c.pendingMsgs[scommon.MsgMetaBlock]
		logger.Log.Debug(context.Background(), c.from, "got MetaBlock from chan")
		// c.GotBlock = true
	}

	c.cachedMetaBlock = msg

	metaBlock := msg.Data.(*mtypes.MetaBlock)
	txs = [][]byte{}
	hashes = [][]byte{}
	if metaBlock != nil {
		txs = metaBlock.Txs
		hashes = make([][]byte, len(metaBlock.Hashlist))
		for i, h := range metaBlock.Hashlist {
			hashes[i] = h.Bytes()
		}
	}
	logger.Log.Debug(context.Background(), c.from, "return Reap", logger.F("hashes", len(hashes)))

	return
}
func (c *Consensus) AddToMempool(txs [][]byte, src string) {
	logger.Log.Info(context.Background(), c.from, "AddToMempool", logger.F("txs", len(txs)))
	groups := c.parseGroups(txs)
	for i := range groups {
		if len(groups[i]) > 0 {
			c.sender.Send(scommon.MsgExtTxBlocks, &types.IncomingTxs{
				Txs:       groups[i],
				Src:       types.NewTxSource(types.TxSourceConsensus, src),
				RequestID: "",
			}, c.height, c.from)

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
	c.height = uint64(height)

	logger.Log.Info(context.Background(), c.from, "ApplyTxsSync")

	if !c.isInited {
		c.isInited = true
	} else {
		c.sender.Send(scommon.MsgExtBlockCompleted, scommon.MsgBlockCompleted_Success, uint64(height-1), c.from)
	}

	// var na int
	txID := fmt.Sprintf("%d", height)

	c.sender.SendSync("transactionalstore", "BeginTransaction", txID, c.height, c.from)

	reapHashlist := make([]evmCommon.Hash, len(hashes))
	for i, h := range hashes {
		reapHashlist[i] = evmCommon.BytesToHash(h)
	}
	logger.Log.Info(context.Background(), c.from, "start send reapinglist", logger.F("reapinglist hashes length", len(reapHashlist)))
	c.sender.Send(scommon.MsgExtReapingList, &types.ReapingList{
		List:      reapHashlist,
		Timestamp: big.NewInt(0),
	}, c.height, c.from)

	logger.Log.Info(context.Background(), c.from, "send reapinglist")

	coinbaseAddress := evmCommon.BytesToAddress(coinbase)
	multiResult := big.NewInt(0).Mul(big.NewInt(timestamp.Unix()), big.NewInt(c.rate))
	blockstamp := big.NewInt(0).Add(big.NewInt(c.starter), multiResult)
	c.sender.Send(scommon.MsgExtBlockStart, &actor.BlockStart{
		Timestamp: blockstamp,
		Coinbase:  coinbaseAddress,
		Height:    uint64(height),
	}, c.height, c.from)
	logger.Log.Info(context.Background(), c.from, "block start")

	// c.AddLog(log.LogLevel_Debug, "[ApplyTxsSync] Before got block.")
	// if !c.GotBlock {
	// 	<-c.pendingMsgs[actor.MsgMetaBlock]
	// }
	// c.AddLog(log.LogLevel_Debug, "[ApplyTxsSync] After got block.")
	// c.GotBlock = false

	msg := <-c.pendingMsgs[scommon.MsgExtAppHash]
	c.sender.SendSync("transactionalstore", "EndTransaction", "", c.height, c.from)
	logger.Log.Debug(context.Background(), c.from, "[ApplyTxsSync] After got apphash.")

	c.sender.Send(scommon.MsgExtBlockEnd, "", c.height, c.from)
	c.sender.Send(scommon.MsgExtReapCommand, "", c.height, c.from)
	return msg.Data.([]byte)
}

func (c *Consensus) GetLocalTxsChan() chan [][]byte {
	return c.chanTxs
}

func (c *Consensus) GetTxsOnBlock(height uint64) ([][]byte, error) {
	request := mtypes.QueryRequest{
		QueryType: mtypes.QueryType_RawBlock,
		Data:      height,
	}

	response, err := c.sender.SendSync("storage", "Query", &request, c.height, c.from)
	if err != nil {
		return nil, err
	}

	return response.(*mtypes.QueryResult).Data.(*mtypes.MonacoBlock).Txs, nil
}

func (c *Consensus) CreateBlockStore() monaco.BlockStore {
	return newBlockStore("tmblockstore", c.sender)
}

func (c *Consensus) CreateStateStore() interface{} {
	return newStateStore("tmstatestore", c.sender)
}

func (c *Consensus) UpdateMaxPeerHeight(height uint64) {
	c.sender.Send(scommon.MsgConsensusMaxPeerHeight, height, height, c.from)
}

func (c *Consensus) SwitchToConsensus() {
	c.sender.Send(scommon.MsgConsensusUp, "", c.height, c.from)
}

func (c *Consensus) startConsensus(backend monaco.BackendProxy, config *config.Config) error {
	logname := "consensus.log"
	rootDir := viper.GetString(cli.HomeFlag)
	//create logger
	if err := tmos.EnsureDir(path.Join(rootDir, "log"), 0777); err != nil {
		panic(err.Error())
		// tmos.PanicSanity(err.Error())
	}
	logfile, err := os.OpenFile(path.Join(rootDir, "log", logname), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		panic(err.Error())
	}

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
