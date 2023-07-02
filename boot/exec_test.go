//go:build !CI

package boot

import (
	"fmt"
	"math"
	"math/big"

	"github.com/arcology-network/concurrenturl/commutative"
	"github.com/arcology-network/concurrenturl/interfaces"
	"github.com/arcology-network/concurrenturl/noncommutative"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/consensus"
	evmtypes "github.com/arcology-network/evm/core/types"
	"github.com/arcology-network/evm/core/vm"
	"github.com/arcology-network/evm/params"
	adaptor "github.com/arcology-network/vm-adaptor"
	"github.com/holiman/uint256"
)

var (
	db          interfaces.Datastore
	coinbase    = evmCommon.BytesToAddress([]byte("coinbase"))
	coreAddress evmCommon.Address
)

/*
	func TestExecSvc(t *testing.T) {
		config.DownloaderCreator = kafka.NewDownloaderCreator(t)
		config.UploaderCreator = kafka.NewUploaderCreator(t)
		//exec.SnapshotLookup = &mockSnapshotDict{}
		intf.RPCCreator = rpc.NewRPCServerInitializer(t)

		globalConfig := config.LoadGlobalConfig("../config/global.json")
		kafkaConfig := config.LoadKafkaConfig("../config/kafka.json")
		appConfig := config.LoadAppConfig("../modules/exec/exec.json")
		brk, _, uploaders := initApp(globalConfig, kafkaConfig, appConfig)

		initDB(t)

		broker := &actor.MessageWrapper{
			MsgBroker:      brk,
			LatestMessage:  actor.NewMessage(),
			WorkThreadName: "unittester",
		}
		coinbase := evmCommon.BytesToAddress([]byte("coinbase"))
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
		contractAddress := evmCommon.BytesToAddress(coreAddress.Bytes())
		for i := 0; i < nBatches; i++ {
			msgs := make([]*cmntypes.StandardMessage, 0, nMsgs)
			//ids := make([]uint32, 0, nMsgs)
			for j := 0; j < nMsgs; j++ {
				data := append(sig, evmCommon.BytesToHash([]byte{byte(i / 256), byte(i % 256), byte(j / 256), byte(j % 256)}).Bytes()...)
				data = append(data, evmCommon.BytesToHash([]byte{byte(i / 256), byte(i % 256), byte(j / 256), byte(j % 256)}).Bytes()...)
				msg := core.NewMessage(
					evmCommon.BytesToAddress(cooAddress.Bytes()),
					&contractAddress,
					0,
					new(big.Int).SetUint64(0),
					10000000,
					new(big.Int).SetUint64(1),
					data,
					nil,
					false,
				)
				msgs = append(msgs, &cmntypes.StandardMessage{
					TxHash: evmCommon.BytesToHash([]byte{byte(i), byte(j / 256), byte(j % 256)}),
					Native: &msg,
				})
			}

			durations := []time.Duration{}
			for i := 0; i < nBatches; i += 4 {
				begin := time.Now()
				response := cmntypes.ExecutorResponses{}
				request := cmntypes.ExecutorRequest{
					Sequences:     []*cmntypes.ExecutingSequence{batches[i], batches[i+1], batches[i+2], batches[i+3]},
					Precedings:    [][]*evmCommon.Hash{},
					PrecedingHash: []evmCommon.Hash{},
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

func (snapshotDict *mockSnapshotDict) Reset(_ *interfaces.Datastore) {}

func (snapshotDict *mockSnapshotDict) AddItem(_ evmCommon.Hash, _ int, _ *interfaces.Datastore) {
}

	func (snapshotDict *mockSnapshotDict) Query(_ []*evmCommon.Hash) (*interfaces.Datastore, []*evmCommon.Hash) {
		return &db, []*evmCommon.Hash{}
	}

func initDB(t *testing.T) {

		persistentDB := ccdb.NewDataStore()
		// meta, _ := commutative.NewMeta(concurrenturl.NewPlatform().Eth10Account())
		persistentDB.Inject(concurrenturl.NewPlatform().Eth10Account(), commutative.NewPath())
		db = urldb.NewTransientDB(persistentDB)

		url := concurrenturl.NewConcurrentUrl(db)
		api := ccapi.NewAPI(url)
		statedb := eth.NewImplStateDB(api)
		// api := adaptor.NewAPIV2(db, url)
		// statedb := adaptor.NewStateDBV2(api, db, url)
		statedb.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
		statedb.CreateAccount(coinbase)
		statedb.CreateAccount(ceoAddress)
		statedb.AddBalance(ceoAddress, new(big.Int).SetUint64(1e18))
		statedb.CreateAccount(cooAddress)
		statedb.AddBalance(cooAddress, new(big.Int).SetUint64(1e18))
		statedb.CreateAccount(cfoAddress)
		statedb.AddBalance(cfoAddress, new(big.Int).SetUint64(1e18))
		_, transitions := url.ExportAll()

		// Deploy KittyCore.
		eu, config := testtools.Prepare(db, 10000000, transitions, []uint32{0})
		transitions, receipt, err := testtools.Deploy(eu, config, ceoAddress, 0, coreCode)
		if err != nil {
			fmt.Printf("Deploy KittyCore err:%v\n", err)
			return
		}
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)
		coreAddress = receipt.ContractAddress
		t.Log(coreAddress)

		// Deploy SaleClockAuction.
		eu, config = testtools.Prepare(db, 10000001, transitions, []uint32{1})
		transitions, receipt, err = testtools.Deploy(eu, config, ceoAddress, 1, saleCode, coreAddress.Bytes(), []byte{100})
		if err != nil {
			fmt.Printf("Deploy SaleClockAuction err:%v\n", err)
			return
		}
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)
		saleAddress := receipt.ContractAddress
		t.Log(saleAddress)

		// Deploy SiringClockAuction.
		eu, config = testtools.Prepare(db, 10000002, transitions, []uint32{2})
		transitions, receipt, err = testtools.Deploy(eu, config, ceoAddress, 2, sireCode, coreAddress.Bytes(), []byte{100})
		if err != nil {
			fmt.Printf("Deploy SiringClockAuction err:%v\n", err)
			return
		}
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)
		sireAddress := receipt.ContractAddress
		t.Log(sireAddress)

		// Deploy GeneScience.
		eu, config = testtools.Prepare(db, 10000003, transitions, []uint32{3})
		transitions, receipt, err = testtools.Deploy(eu, config, ceoAddress, 3, geneCode, []byte{}, coreAddress.Bytes())
		if err != nil {
			fmt.Printf("Deploy GeneScience err:%v\n", err)
			return
		}
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)
		geneAddress := receipt.ContractAddress
		t.Log(geneAddress)

		// Call setSaleAuctionAddress.
		eu, config = testtools.Prepare(db, 10000004, transitions, []uint32{4})
		transitions, receipt = testtools.Run(eu, config, &ceoAddress, &coreAddress, 4, true, "setSaleAuctionAddress(address)", saleAddress.Bytes())
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)

		// Call setSiringAuctionAddress.
		eu, config = testtools.Prepare(db, 10000005, transitions, []uint32{5})
		transitions, receipt = testtools.Run(eu, config, &ceoAddress, &coreAddress, 5, true, "setSiringAuctionAddress(address)", sireAddress.Bytes())
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)

		// Call setGeneScienceAddress.
		eu, config = testtools.Prepare(db, 10000006, transitions, []uint32{6})
		transitions, receipt = testtools.Run(eu, config, &ceoAddress, &coreAddress, 6, true, "setGeneScienceAddress(address)", geneAddress.Bytes())
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)

		// Call setCOO.
		eu, config = testtools.Prepare(db, 10000007, transitions, []uint32{7})
		transitions, receipt = testtools.Run(eu, config, &ceoAddress, &coreAddress, 7, true, "setCOO(address)", cooAddress.Bytes())
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)

		// Call setCFO.
		eu, config = testtools.Prepare(db, 10000008, transitions, []uint32{8})
		transitions, receipt = testtools.Run(eu, config, &ceoAddress, &coreAddress, 8, true, "setCFO(address)", cfoAddress.Bytes())
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)

		// Call unpause.
		eu, config = testtools.Prepare(db, 10000009, transitions, []uint32{9})
		transitions, receipt = testtools.Run(eu, config, &ceoAddress, &coreAddress, 9, true, "unpause()")
		t.Log("\n" + formatTransitions(transitions))
		t.Log(receipt)
	}
*/
func formatValue(value interface{}) string {
	switch value.(type) {
	case *commutative.Path:
		meta := value.(*commutative.Path)
		var str string
		str += "{"
		for i, k := range meta.Keys() {
			str += k
			if i != len(meta.Keys())-1 {
				str += ", "
			}
		}
		str += "}"
		if len(meta.Added()) != 0 {
			str += " + {"
			for i, k := range meta.Added() {
				str += k
				if i != len(meta.Added())-1 {
					str += ", "
				}
			}
			str += "}"
		}
		if len(meta.Removed()) != 0 {
			str += " - {"
			for i, k := range meta.Removed() {
				str += k
				if i != len(meta.Removed())-1 {
					str += ", "
				}
			}
			str += "}"
		}
		return str
	case *noncommutative.Int64:
		return fmt.Sprintf(" = %v", int64(*value.(*noncommutative.Int64)))
	case *noncommutative.Bytes:
		return fmt.Sprintf(" = %v", value.(*noncommutative.Bytes).Value())
	case *commutative.U256:
		v := value.(*commutative.U256).Value()
		d := value.(*commutative.U256).Delta()
		return fmt.Sprintf(" = %v + %v", v.(*uint256.Int).ToBig(), d.(*uint256.Int).ToBig())
	case *commutative.Int64:
		v := value.(*commutative.Int64).Value()
		d := value.(*commutative.Int64).Value()
		return fmt.Sprintf(" = %v + %v", v, d)
	}
	return ""
}

func formatTransitions(transitions []interfaces.Univalue) string {
	var str string
	for _, t := range transitions {
		str += fmt.Sprintf("[%v:%v,%v,%v]%s%s\n", t.(interfaces.Univalue).GetTx(), t.(interfaces.Univalue).Reads(), t.(interfaces.Univalue).Writes(), t.(interfaces.Univalue).Preexist(), (*t.(interfaces.Univalue).GetPath()), formatValue(t.(interfaces.Univalue).Value()))
	}
	return str
}

// fakeChain implements the ChainContext interface.
type fakeChain struct {
}

func (chain *fakeChain) GetHeader(evmCommon.Hash, uint64) *evmtypes.Header {
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
		ParentHash:  evmCommon.Hash{},
		Time:        new(big.Int).SetUint64(10000000),
		Coinbase:    &coinbase,
		GasLimit:    math.MaxUint64,
		Difficulty:  new(big.Int).SetUint64(10000000),
	}
	cfg.Chain = new(fakeChain)
	return cfg
}
