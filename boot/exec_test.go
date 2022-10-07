package boot

import (
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	ccdb "github.com/HPISTechnologies/common-lib/cachedstorage"
	cmntypes "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/mock/kafka"
	"github.com/HPISTechnologies/component-lib/mock/rpc"
	"github.com/HPISTechnologies/concurrenturl/v2"
	urlcommon "github.com/HPISTechnologies/concurrenturl/v2/common"
	urldb "github.com/HPISTechnologies/concurrenturl/v2/storage"
	urltype "github.com/HPISTechnologies/concurrenturl/v2/type"
	"github.com/HPISTechnologies/concurrenturl/v2/type/commutative"
	"github.com/HPISTechnologies/concurrenturl/v2/type/noncommutative"
	evmcommon "github.com/HPISTechnologies/evm/common"
	"github.com/HPISTechnologies/evm/consensus"
	evmtypes "github.com/HPISTechnologies/evm/core/types"
	"github.com/HPISTechnologies/evm/core/vm"
	"github.com/HPISTechnologies/evm/crypto"
	"github.com/HPISTechnologies/evm/params"
	"github.com/HPISTechnologies/main/config"
	"github.com/HPISTechnologies/main/modules/exec"
	adaptor "github.com/HPISTechnologies/vm-adaptor/evm"
)

var (
	db          urlcommon.DatastoreInterface
	coinbase    = evmcommon.BytesToAddress([]byte("coinbase"))
	coreAddress evmcommon.Address
)

func TestExecSvc(t *testing.T) {
	config.DownloaderCreator = kafka.NewDownloaderCreator(t)
	config.UploaderCreator = kafka.NewUploaderCreator(t)
	exec.SnapshotLookup = &mockSnapshotDict{}
	intf.RPCCreator = rpc.NewRPCServerInitializer(t)

	globalConfig := config.LoadGlobalConfig("../config/global.json")
	kafkaConfig := config.LoadKafkaConfig("../config/kafka.json")
	appConfig := config.LoadAppConfig("../config/exec.json")
	brk, _, uploaders := initApp(globalConfig, kafkaConfig, appConfig)

	initDB(t)

	broker := &actor.MessageWrapper{
		MsgBroker:      brk,
		LatestMessage:  actor.NewMessage(),
		WorkThreadName: "unittester",
	}
	coinbase := ethCommon.BytesToAddress([]byte("coinbase"))
	broker.Send(actor.MsgBlockCompleted, actor.MsgBlockCompleted_Success)
	broker.Send(actor.MsgBlockStart, &actor.BlockStart{
		Coinbase:  coinbase,
		Height:    1,
		Timestamp: new(big.Int).SetUint64(10000),
	}, 1)
	broker.Send(actor.MsgParentInfo, &cmntypes.ParentInfo{}, 1)

	nBatches := 4
	nMsgs := 500
	batches := make([]*cmntypes.ExecutingSequence, 0, nBatches)
	sig := crypto.Keccak256([]byte("createPromoKitty(uint256,address)"))[:4]
	contractAddress := ethCommon.BytesToAddress(coreAddress.Bytes())
	for i := 0; i < nBatches; i++ {
		msgs := make([]*cmntypes.StandardMessage, 0, nMsgs)
		ids := make([]uint32, 0, nMsgs)
		for j := 0; j < nMsgs; j++ {
			data := append(sig, evmcommon.BytesToHash([]byte{byte(i / 256), byte(i % 256), byte(j / 256), byte(j % 256)}).Bytes()...)
			data = append(data, evmcommon.BytesToHash([]byte{byte(i / 256), byte(i % 256), byte(j / 256), byte(j % 256)}).Bytes()...)
			msg := ethTypes.NewMessage(
				ethCommon.BytesToAddress(cooAddress.Bytes()),
				&contractAddress,
				0,
				new(big.Int).SetUint64(0),
				10000000,
				new(big.Int).SetUint64(1),
				data,
				false,
			)
			msgs = append(msgs, &cmntypes.StandardMessage{
				TxHash: ethCommon.BytesToHash([]byte{byte(i), byte(j / 256), byte(j % 256)}),
				Native: &msg,
			})
			ids = append(ids, uint32(i*nMsgs+j+1))
		}
		batches = append(batches, &cmntypes.ExecutingSequence{
			Msgs:     msgs,
			Txids:    ids,
			Parallel: true,
		})
	}

	durations := []time.Duration{}
	for i := 0; i < nBatches; i += 4 {
		begin := time.Now()
		response := cmntypes.ExecutorResponses{}
		request := cmntypes.ExecutorRequest{
			Sequences:     []*cmntypes.ExecutingSequence{batches[i], batches[i+1], batches[i+2], batches[i+3]},
			Precedings:    []*ethCommon.Hash{},
			PrecedingHash: ethCommon.Hash{},
			Timestamp:     new(big.Int),
			Parallelism:   4,
		}

		intf.Router.Call("executor-1", "ExecTxs", &actor.Message{
			Name:   actor.MsgTxsToExecute,
			Height: 1,
			Data:   &request,
		}, &response)
		durations = append(durations, time.Since(begin))

		for _, status := range response.StatusList {
			if status != 1 {
				t.Fail()
				return
			}
		}
	}

	t.Log(formatDurations(durations))
	time.Sleep(3 * time.Second)
	t.Log(uploaders[0].(*kafka.Uploader).GetCounter())
	t.Log(uploaders[0].(*kafka.Uploader).GetDataSizeCounter())
	t.Log(uploaders[1].(*kafka.Uploader).GetCounter())
	t.Log(uploaders[1].(*kafka.Uploader).GetDataSizeCounter())
	t.Log(uploaders[2].(*kafka.Uploader).GetCounter())
	t.Log(uploaders[2].(*kafka.Uploader).GetDataSizeCounter())
}

func formatDurations(durations []time.Duration) string {
	str := ""
	for _, d := range durations {
		str += fmt.Sprintf("%d,", d.Milliseconds())
	}
	return str
}

type mockSnapshotDict struct {
}

func (snapshotDict *mockSnapshotDict) Reset(_ *urlcommon.DatastoreInterface) {}

func (snapshotDict *mockSnapshotDict) AddItem(_ ethCommon.Hash, _ int, _ *urlcommon.DatastoreInterface) {
}

func (snapshotDict *mockSnapshotDict) Query(_ []*ethCommon.Hash) (*urlcommon.DatastoreInterface, []*ethCommon.Hash) {
	return &db, []*ethCommon.Hash{}
}

func initDB(t *testing.T) {

	persistentDB := ccdb.NewDataStore()
	meta, _ := commutative.NewMeta(urlcommon.NewPlatform().Eth10Account())
	persistentDB.Inject(urlcommon.NewPlatform().Eth10Account(), meta)
	db = urldb.NewTransientDB(persistentDB)

	url := concurrenturl.NewConcurrentUrl(db)
	api := adaptor.NewAPIV2(db, url)
	statedb := adaptor.NewStateDBV2(api, db, url)
	statedb.Prepare(evmcommon.Hash{}, evmcommon.Hash{}, 0)
	statedb.CreateAccount(coinbase)
	statedb.CreateAccount(ceoAddress)
	statedb.AddBalance(ceoAddress, new(big.Int).SetUint64(1e18))
	statedb.CreateAccount(cooAddress)
	statedb.AddBalance(cooAddress, new(big.Int).SetUint64(1e18))
	statedb.CreateAccount(cfoAddress)
	statedb.AddBalance(cfoAddress, new(big.Int).SetUint64(1e18))
	_, transitions := url.Export(true)

	// Deploy KittyCore.
	eu, config := prepare(db, 10000000, transitions, []uint32{0})
	transitions, receipt := deploy(eu, config, ceoAddress, 0, coreCode)
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)
	coreAddress = receipt.ContractAddress
	t.Log(coreAddress)

	// Deploy SaleClockAuction.
	eu, config = prepare(db, 10000001, transitions, []uint32{1})
	transitions, receipt = deploy(eu, config, ceoAddress, 1, saleCode, coreAddress.Bytes(), []byte{100})
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)
	saleAddress := receipt.ContractAddress
	t.Log(saleAddress)

	// Deploy SiringClockAuction.
	eu, config = prepare(db, 10000002, transitions, []uint32{2})
	transitions, receipt = deploy(eu, config, ceoAddress, 2, sireCode, coreAddress.Bytes(), []byte{100})
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)
	sireAddress := receipt.ContractAddress
	t.Log(sireAddress)

	// Deploy GeneScience.
	eu, config = prepare(db, 10000003, transitions, []uint32{3})
	transitions, receipt = deploy(eu, config, ceoAddress, 3, geneCode, []byte{}, coreAddress.Bytes())
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)
	geneAddress := receipt.ContractAddress
	t.Log(geneAddress)

	// Call setSaleAuctionAddress.
	eu, config = prepare(db, 10000004, transitions, []uint32{4})
	transitions, receipt = run(eu, config, &ceoAddress, &coreAddress, 4, true, "setSaleAuctionAddress(address)", saleAddress.Bytes())
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)

	// Call setSiringAuctionAddress.
	eu, config = prepare(db, 10000005, transitions, []uint32{5})
	transitions, receipt = run(eu, config, &ceoAddress, &coreAddress, 5, true, "setSiringAuctionAddress(address)", sireAddress.Bytes())
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)

	// Call setGeneScienceAddress.
	eu, config = prepare(db, 10000006, transitions, []uint32{6})
	transitions, receipt = run(eu, config, &ceoAddress, &coreAddress, 6, true, "setGeneScienceAddress(address)", geneAddress.Bytes())
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)

	// Call setCOO.
	eu, config = prepare(db, 10000007, transitions, []uint32{7})
	transitions, receipt = run(eu, config, &ceoAddress, &coreAddress, 7, true, "setCOO(address)", cooAddress.Bytes())
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)

	// Call setCFO.
	eu, config = prepare(db, 10000008, transitions, []uint32{8})
	transitions, receipt = run(eu, config, &ceoAddress, &coreAddress, 8, true, "setCFO(address)", cfoAddress.Bytes())
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)

	// Call unpause.
	eu, config = prepare(db, 10000009, transitions, []uint32{9})
	transitions, receipt = run(eu, config, &ceoAddress, &coreAddress, 9, true, "unpause()")
	t.Log("\n" + formatTransitions(transitions))
	t.Log(receipt)
}

func prepare(db urlcommon.DatastoreInterface, height uint64, transitions []urlcommon.UnivalueInterface, txs []uint32) (*adaptor.EUV2, *adaptor.Config) {
	url := concurrenturl.NewConcurrentUrl(db)
	url.AllInOneCommit(transitions, txs)
	api := adaptor.NewAPIV2(db, url)
	statedb := adaptor.NewStateDBV2(api, db, url)

	config := MainConfig()
	config.Coinbase = &coinbase
	config.BlockNumber = new(big.Int).SetUint64(height)
	config.Time = new(big.Int).SetUint64(height)

	return adaptor.NewEUV2(config.ChainConfig, *config.VMConfig, config.Chain, statedb, api, db, url), config
}

func deploy(eu *adaptor.EUV2, config *adaptor.Config, owner evmcommon.Address, nonce uint64, code string, args ...[]byte) ([]urlcommon.UnivalueInterface, *evmtypes.Receipt) {
	data := evmcommon.Hex2Bytes(code)
	for _, arg := range args {
		data = append(data, evmcommon.BytesToHash(arg).Bytes()...)
	}
	msg := evmtypes.NewMessage(owner, nil, nonce, new(big.Int).SetUint64(0), 1e15, new(big.Int).SetUint64(1), data, nil, true)
	_, transitions, receipt := eu.Run(evmcommon.BytesToHash([]byte{byte(nonce + 1), byte(nonce + 1), byte(nonce + 1)}), int(nonce+1), &msg, adaptor.NewEVMBlockContextV2(config), adaptor.NewEVMTxContext(msg))
	return transitions, receipt
}

func run(eu *adaptor.EUV2, config *adaptor.Config, from, to *evmcommon.Address, nonce uint64, checkNonce bool, function string, args ...[]byte) ([]urlcommon.UnivalueInterface, *evmtypes.Receipt) {
	data := crypto.Keccak256([]byte(function))[:4]
	for _, arg := range args {
		data = append(data, evmcommon.BytesToHash(arg).Bytes()...)
	}
	msg := evmtypes.NewMessage(*from, to, nonce, new(big.Int).SetUint64(0), 1e15, new(big.Int).SetUint64(1), data, nil, checkNonce)
	_, transitions, receipt := eu.Run(evmcommon.BytesToHash([]byte{byte(nonce + 1), byte(nonce + 1), byte(nonce + 1)}), int(nonce+1), &msg, adaptor.NewEVMBlockContextV2(config), adaptor.NewEVMTxContext(msg))
	return transitions, receipt
}

func formatValue(value interface{}) string {
	switch value.(type) {
	case *commutative.Meta:
		meta := value.(*commutative.Meta)
		var str string
		str += "{"
		for i, k := range meta.PeekKeys() {
			str += k
			if i != len(meta.PeekKeys())-1 {
				str += ", "
			}
		}
		str += "}"
		if len(meta.PeekAdded()) != 0 {
			str += " + {"
			for i, k := range meta.PeekAdded() {
				str += k
				if i != len(meta.PeekAdded())-1 {
					str += ", "
				}
			}
			str += "}"
		}
		if len(meta.PeekRemoved()) != 0 {
			str += " - {"
			for i, k := range meta.PeekRemoved() {
				str += k
				if i != len(meta.PeekRemoved())-1 {
					str += ", "
				}
			}
			str += "}"
		}
		return str
	case *noncommutative.Int64:
		return fmt.Sprintf(" = %v", int64(*value.(*noncommutative.Int64)))
	case *noncommutative.Bytes:
		return fmt.Sprintf(" = %v", value.(*noncommutative.Bytes).Data())
	case *commutative.Balance:
		v := value.(*commutative.Balance).Value()
		d := value.(*commutative.Balance).GetDelta().(*big.Int)
		return fmt.Sprintf(" = %v + %v", v.(*big.Int).Uint64(), d.Int64())
	case *commutative.Int64:
		v := value.(*commutative.Int64).Value()
		d := value.(*commutative.Int64).GetDelta()
		return fmt.Sprintf(" = %v + %v", v, d)
	}
	return ""
}

func formatTransitions(transitions []urlcommon.UnivalueInterface) string {
	var str string
	for _, t := range transitions {
		str += fmt.Sprintf("[%v:%v,%v,%v,%v]%s%s\n", t.(*urltype.Univalue).GetTx(), t.(*urltype.Univalue).Reads(), t.(*urltype.Univalue).Writes(), t.(*urltype.Univalue).Preexist(), t.(*urltype.Univalue).Composite(), (*t.(*urltype.Univalue).GetPath()), formatValue(t.(*urltype.Univalue).Value()))
	}
	return str
}

// fakeChain implements the ChainContext interface.
type fakeChain struct {
}

func (chain *fakeChain) GetHeader(evmcommon.Hash, uint64) *evmtypes.Header {
	return &evmtypes.Header{}
}

func (chain *fakeChain) Engine() consensus.Engine {
	return nil
}

func MainConfig() *adaptor.Config {
	vmConfig := vm.Config{}
	cfg := &adaptor.Config{
		ChainConfig: params.MainnetChainConfig,
		VMConfig:    &vmConfig,
		BlockNumber: new(big.Int).SetUint64(10000000),
		ParentHash:  evmcommon.Hash{},
		Time:        new(big.Int).SetUint64(10000000),
		Coinbase:    &coinbase,
		GasLimit:    math.MaxUint64,
		Difficulty:  new(big.Int).SetUint64(10000000),
	}
	cfg.Chain = new(fakeChain)
	return cfg
}
