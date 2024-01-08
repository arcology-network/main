package storage

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/transactional"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/storage"
	ccurl "github.com/arcology-network/concurrenturl"
	ccurlcommon "github.com/arcology-network/concurrenturl/common"
	"github.com/arcology-network/concurrenturl/commutative"
	"github.com/arcology-network/concurrenturl/interfaces"
	ccdb "github.com/arcology-network/concurrenturl/storage"
	"github.com/arcology-network/consensus-engine/state"
	"github.com/arcology-network/main/modules/core"
	"github.com/ethereum/go-ethereum/cmd/utils"
	evmCommon "github.com/ethereum/go-ethereum/common"

	"github.com/arcology-network/common-lib/merkle"
	indexer "github.com/arcology-network/concurrenturl/indexer"
	ccapi "github.com/arcology-network/vm-adaptor/api"
	"github.com/arcology-network/vm-adaptor/eth"

	evmcore "github.com/ethereum/go-ethereum/core"
)

type Initializer struct {
	actor.WorkerThread

	genesisFile     string
	storage_db_path string

	SignerType uint8

	proof *ccdb.MerkleProof
}

func NewInitializer(concurrency int, groupId string) actor.IWorkerEx {
	w := &Initializer{}
	w.Set(concurrency, groupId)
	w.SignerType = types.Signer_London
	return w

}

func (i *Initializer) Inputs() ([]string, bool) {
	return []string{}, false
}

func (i *Initializer) Outputs() map[string]int {
	return map[string]int{
		actor.MsgInitDB:    1,
		actor.MsgStorageUp: 1,
		actor.MsgCoinbase:  1,
	}
}

// Config implements Configurable interface.
func (i *Initializer) Config(params map[string]interface{}) {
	i.genesisFile = params["genesis_file"].(string)
	i.storage_db_path = params["storage_db_path"].(string)
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

	var db interfaces.Datastore
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

		db, rootHash = i.initGenesisAccounts(genesis)

		evmblock := genesis.ToBlock()

		block, err := core.CreateBlock(evmblock.Header(), [][]byte{}, i.SignerType)
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

		// db = ccdb.NewParallelEthMemDataStore()
		db = ccdb.NewLevelDBDataStore(i.storage_db_path)

		db.Inject(RootPrefix, commutative.NewPath())

		// Register recover function.
		transactional.RegisterRecoverFunc("urlupdate", func(_ interface{}, bs []byte) error {
			var updates storage.UrlUpdate
			if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&updates); err != nil {
				fmt.Printf("Error decoding UrlUpdate, err = %v\n", err)
				return err
			}

			values := make([]interface{}, len(updates.EncodedValues))
			for i, v := range updates.EncodedValues {
				values[i] = ccdb.Codec{}.Decode(v, nil) //urltyp.FromBytes(v)
			}
			db.BatchInject(updates.Keys, values)
			fmt.Printf("[storage.Initializer] Recover urlupdate.\n")
			return nil
		})
		transactional.RegisterRecoverFunc("parentinfo", func(_ interface{}, bs []byte) error {
			var pi types.ParentInfo
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

	ddb := db.(*ccdb.EthDataStore)
	proof, err := ccdb.NewMerkleProof(ddb.EthDB(), ddb.Root()) // db.(*ccdb.EthDataStore)
	if err != nil {
		panic("MerkleProof create failed")
	}
	i.proof = proof

	intf.Router.Call("urlstore", "Init", db, &na)

	return []*actor.Message{
		{
			Name:   actor.MsgInitDB,
			Height: uint64(height),
			Data:   db,
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
	}
}

func (i *Initializer) QueryState(ctx context.Context, request *types.QueryRequest, response *types.QueryResult) error {
	switch request.QueryType {
	case types.QueryType_Proof:
		rq := request.Data.(*types.RequestProof)
		keys := make([]string, len(rq.Keys))
		for i := range keys {
			keys[i] = fmt.Sprintf("%x", rq.Keys[i].Bytes())
		}
		result, err := i.proof.GetProof(fmt.Sprintf("%x", rq.Address.Bytes()), keys)
		if err != nil {
			return err
		}
		response.Data = result
	}
	return nil
}

func (i *Initializer) OnStart() {}

func (i *Initializer) OnMessageArrived(msgs []*actor.Message) error {
	return nil
}

func (i *Initializer) initGenesisAccounts(genesis *evmcore.Genesis) (interfaces.Datastore, evmCommon.Hash) {
	// db := ccdb.NewParallelEthMemDataStore()
	db := ccdb.NewLevelDBDataStore(i.storage_db_path)

	db.Inject(RootPrefix, commutative.NewPath())

	transitions := i.createTransitions(db, genesis.Alloc) //i.generateTransitions(db, accounts)
	url := ccurl.NewConcurrentUrl(db)
	url.Import(common.Clone(transitions))
	url.Sort()
	url.Finalize([]uint32{0})
	keys, values := url.KVs()

	encodedValues := make([][]byte, 0, len(values))
	for _, v := range values {
		if v != nil {
			univalue := v.(interfaces.Univalue)
			data := ccdb.Codec{}.Encode("", univalue.Value())
			encodedValues = append(encodedValues, data)
		} else {
			encodedValues = append(encodedValues, []byte{})
		}
	}

	merkle := indexer.NewAccountMerkle(ccurlcommon.NewPlatform(), indexer.RlpEncoder, merkle.Keccak256{}.Hash)
	merkle.Import(common.Clone(transitions))
	rootHash := calcRootHash(merkle, evmCommon.Hash{}, keys, encodedValues)
	url.WriteToDbBuffer()
	url.SaveToDB()
	return db, rootHash
}

//--------------------------------------------------------------------------------------------------------------------------------

func (i *Initializer) createTransitions(db interfaces.Datastore, genesisAlloc evmcore.GenesisAlloc) []interfaces.Univalue {
	batch := 10
	addresses := make([]evmCommon.Address, 0, batch)
	index := 0
	transitions := make([]interfaces.Univalue, 0, len(genesisAlloc)*10)
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

func getTransition(db interfaces.Datastore, addresses []evmCommon.Address, genesisAlloc evmcore.GenesisAlloc) []interfaces.Univalue {
	url := ccurl.NewConcurrentUrl(db)
	api := ccapi.NewAPI(url)
	stateDB := eth.NewImplStateDB(api)
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
	_, transitions := url.ExportAll()

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
