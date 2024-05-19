package storage

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/arcology-network/common-lib/storage/transactional"
	"github.com/arcology-network/consensus-engine/state"
	adaptorcommon "github.com/arcology-network/evm-adaptor/pathbuilder"
	"github.com/arcology-network/main/modules/core"
	"github.com/arcology-network/storage-committer/commutative"
	"github.com/arcology-network/storage-committer/interfaces"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/ethereum/go-ethereum/cmd/utils"
	evmCommon "github.com/ethereum/go-ethereum/common"

	apihandler "github.com/arcology-network/evm-adaptor/apihandler"
	cache "github.com/arcology-network/storage-committer/storage/writecache"
	evmcore "github.com/ethereum/go-ethereum/core"

	"github.com/arcology-network/common-lib/exp/mempool"
	"github.com/arcology-network/common-lib/exp/slice"
	mtypes "github.com/arcology-network/main/types"
	univaluepk "github.com/arcology-network/storage-committer/univalue"

	statestore "github.com/arcology-network/storage-committer"
	stgproxy "github.com/arcology-network/storage-committer/storage/proxy"
)

type Initializer struct {
	actor.WorkerThread

	genesisFile     string
	storage_db_path string
}

func NewInitializer(concurrency int, groupId string) actor.IWorkerEx {
	w := &Initializer{}
	w.Set(concurrency, groupId)
	return w

}

func (i *Initializer) Inputs() ([]string, bool) {
	return []string{}, false
}

func (i *Initializer) Outputs() map[string]int {
	return map[string]int{
		actor.MsgInitDB:      1,
		actor.MsgStorageUp:   1,
		actor.MsgCoinbase:    1,
		actor.MsgChainConfig: 1,
	}
}

// Config implements Configurable interface.
func (i *Initializer) Config(params map[string]interface{}) {
	i.genesisFile = params["genesis_file"].(string)
	i.storage_db_path = params["dbpath"].(string)
}

func (i *Initializer) InitMsgs() []*actor.Message {
	var na int
	var state state.State
	if err := intf.Router.Call("tmstatestore", "Load", &na, &state); err != nil {
		panic(err)
	}
	height := state.LastBlockHeight

	genesis := i.readGenesis(i.genesisFile)
	blockStart := &actor.BlockStart{
		Timestamp: big.NewInt(int64(genesis.Timestamp)),
		Coinbase:  genesis.Coinbase,
		Extra:     genesis.ExtraData,
	}

	var store *statestore.StateStore
	var rootHash evmCommon.Hash
	if height == 0 {
		// Make place holder for recover functions.
		transactional.RegisterRecoverFunc("urlupdate", func(interface{}, []byte) error {
			return nil
		})
		transactional.RegisterRecoverFunc("parentinfo", func(interface{}, []byte) error {
			return nil
		})
		transactional.RegisterRecoverFunc("schdstate", func(interface{}, []byte) error {
			return nil
		})

		store, rootHash = i.initGenesisAccounts(genesis, uint64(height))

		evmblock := genesis.ToBlock()

		block, err := core.CreateBlock(evmblock.Header(), [][]byte{}, mtypes.GetSignerType(big.NewInt(height), genesis.Config))
		if err != nil {
			panic("Create genesis block err!")
		}
		intf.Router.Call("blockstore", "Save", block, &na)
		hash := evmblock.Hash()
		var na int
		intf.Router.Call("statestore", "Save", &State{
			Height:     0,
			ParentHash: hash,
			ParentRoot: rootHash,
		}, &na)

	} else {

		// db = ccdb.NewLevelDBDataStore(i.storage_db_path)

		db := stgproxy.NewLevelDBStoreProxy(i.storage_db_path) //.EnableCache()
		db.Inject(RootPrefix, commutative.NewPath())
		store = statestore.NewStateStore(db)

		// Register recover function.
		transactional.RegisterRecoverFunc("urlupdate", func(_ interface{}, bs []byte) error {
			// var updates storage.UrlUpdate
			// if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&updates); err != nil {
			// 	fmt.Printf("Error decoding UrlUpdate, err = %v\n", err)
			// 	return err
			// }

			// values := make([]interface{}, len(updates.EncodedValues))
			// for i, v := range updates.EncodedValues {
			// 	values[i] = ccdb.Codec{}.Decode(v, nil) //urltyp.FromBytes(v)
			// }

			// db.BatchInject(updates.Keys, values)
			// fmt.Printf("[storage.Initializer] Recover urlupdate.\n")
			return nil
		})
		transactional.RegisterRecoverFunc("parentinfo", func(_ interface{}, bs []byte) error {
			var pi mtypes.ParentInfo
			if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&pi); err != nil {
				fmt.Printf("Error decoding ParentInfo, err = %v\n", err)
				return err
			}

			var na int
			intf.Router.Call("statestore", "Save", &State{
				Height:     uint64(height),
				ParentHash: pi.ParentHash,
				ParentRoot: pi.ParentRoot,
			}, &na)
			fmt.Printf("[storage.Initializer] Recover parentinfo = %v\n", pi)
			return nil
		})
		transactional.RegisterRecoverFunc("schdstate", func(_ interface{}, bs []byte) error {
			var state SchdState
			if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&state); err != nil {
				fmt.Printf("Error decoding SchdState, err = %v\n", err)
				return err
			}

			var na int
			intf.Router.Call("schdstore", "DirectWrite", &state, &na)
			fmt.Printf("[storage.Initializer] Recover schdstate.\n")
			return nil
		})
		// Recover.
		txID := fmt.Sprintf("%d", height)
		var na int
		fmt.Printf("[storage.Initializer] Recover transactional store to height: %s\n", txID)
		err := intf.Router.Call("transactionalstore", "Recover", &txID, &na)
		if err != nil {
			panic(fmt.Sprintf("[storage.Initializer] Error occurred while recovering transactional store: %v\n", err))
		}
	}

	intf.Router.Call("urlstore", "Init", store, &na)

	return []*actor.Message{
		{
			Name:   actor.MsgInitDB,
			Height: uint64(height),
			Data:   store,
		},
		{
			Name:   actor.MsgStorageUp,
			Height: uint64(height),
		},
		{
			Name:   actor.MsgCoinbase,
			Height: uint64(height),
			Data:   blockStart,
		},
		{
			Name:   actor.MsgChainConfig,
			Height: uint64(height),
			Data:   genesis.Config,
		},
	}
}

func (i *Initializer) OnStart() {}

func (i *Initializer) OnMessageArrived(msgs []*actor.Message) error {
	return nil
}

func (i *Initializer) initGenesisAccounts(genesis *evmcore.Genesis, height uint64) (*statestore.StateStore, evmCommon.Hash) {
	db := stgproxy.NewLevelDBStoreProxy(i.storage_db_path)
	stateStore := statestore.NewStateStore(db)
	db.Inject(RootPrefix, commutative.NewPath())

	transitions := i.createTransitions(db, genesis.Alloc)

	stateStore.Import(slice.Clone(transitions))
	stateStore.Precommit([]uint32{0})
	stateStore.Commit(height)

	return stateStore, evmCommon.Hash{}
}

//--------------------------------------------------------------------------------------------------------------------------------

func (i *Initializer) createTransitions(db interfaces.ReadOnlyStore, genesisAlloc evmcore.GenesisAlloc) []*univaluepk.Univalue {
	batch := 10
	addresses := make([]evmCommon.Address, 0, batch)
	index := 0
	transitions := make([]*univaluepk.Univalue, 0, len(genesisAlloc)*10)
	for addr, _ := range genesisAlloc {
		if index%batch == 0 && index > 0 {
			transitions = append(transitions, getTransition(db, addresses, genesisAlloc)...)
			addresses = make([]evmCommon.Address, 0, batch)
		}
		addresses = append(addresses, addr)
		index++
	}
	if len(addresses) > 0 {
		transitions = append(transitions, getTransition(db, addresses, genesisAlloc)...)
	}
	return transitions
}

func getTransition(db interfaces.ReadOnlyStore, addresses []evmCommon.Address, genesisAlloc evmcore.GenesisAlloc) []*univaluepk.Univalue {
	api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
		return cache.NewWriteCache(db, 32, 1)
	}, func(cache *cache.WriteCache) { cache.Clear() }))

	stateDB := adaptorcommon.NewImplStateDB(api)
	stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for _, addr := range addresses {
		acct := genesisAlloc[addr]
		stateDB.CreateAccount(addr)
		stateDB.SetBalance(addr, acct.Balance)
		stateDB.SetNonce(addr, uint64(1))
		code := acct.Code
		if len(code) > 0 {
			stateDB.SetCode(addr, code)
		}
		for k, v := range acct.Storage {
			stateDB.SetState(addr, k, v)
		}

	}
	_, transitions := api.WriteCache().(*cache.WriteCache).ExportAll()

	return transitions
}

// readGenesis will read the given JSON format genesis file and return
// the initialized Genesis structure
func (i *Initializer) readGenesis(genesisPath string) *evmcore.Genesis {
	// Make sure we have a valid genesis JSON
	//genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		utils.Fatalf("Must supply path to genesis JSON file")
	}
	file, err := os.Open(genesisPath)
	if err != nil {
		utils.Fatalf("Failed to read genesis file: %v", err)
	}
	defer file.Close()

	genesis := new(evmcore.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		utils.Fatalf("invalid genesis file: %v", err)
	}
	return genesis
}
