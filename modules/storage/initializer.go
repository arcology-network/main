package storage

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/transactional"
	types "github.com/arcology-network/common-lib/types"
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
	evmCommon "github.com/ethereum/go-ethereum/common"

	"github.com/arcology-network/common-lib/merkle"
	indexer "github.com/arcology-network/concurrenturl/indexer"
	ccapi "github.com/arcology-network/vm-adaptor/api"
	"github.com/arcology-network/vm-adaptor/eth"
)

type Initializer struct {
	actor.WorkerThread

	accountFile      string
	storage_url_path string
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
		actor.MsgInitDB:    1,
		actor.MsgStorageUp: 1,
	}
}

// Config implements Configurable interface.
func (i *Initializer) Config(params map[string]interface{}) {
	//mstypes.CreateDB(params)
	i.accountFile = params["account_file"].(string)
	i.storage_url_path = params["storage_url_path"].(string)
}

func (i *Initializer) InitMsgs() []*actor.Message {
	var na int
	var state state.State
	if err := intf.Router.Call("tmstatestore", "Load", &na, &state); err != nil {
		panic(err)
	}
	height := state.LastBlockHeight

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

		db, rootHash = i.initGenesisAccounts()
		parentinfo := &types.ParentInfo{
			ParentHash: evmCommon.Hash{},
			ParentRoot: evmCommon.Hash{},
		}
		header, block, err := core.CreateBlock(parentinfo, uint64(0), big.NewInt(0), evmCommon.Address{}, rootHash, uint64(0), evmCommon.Hash{}, evmCommon.Hash{}, [][]byte{})
		if err != nil {
			panic("Create genesis block err!")
		}
		intf.Router.Call("blockstore", "Save", block, &na)

		var na int
		intf.Router.Call("statestore", "Save", &State{
			Height:     0,
			ParentHash: header.Hash(),
			ParentRoot: rootHash,
		}, &na)
		parentinfo = &types.ParentInfo{
			ParentHash: header.Hash(),
			ParentRoot: rootHash,
		}

	} else {

		// db = ccdb.NewParallelEthMemDataStore()
		db = ccdb.NewLevelDBDataStore(i.storage_url_path)

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
	}
}

func (i *Initializer) OnStart() {}

func (i *Initializer) OnMessageArrived(msgs []*actor.Message) error {
	return nil
}

func (i *Initializer) initGenesisAccounts() (interfaces.Datastore, evmCommon.Hash) {
	accounts := i.loadGenesisAccounts()

	// db := ccdb.NewParallelEthMemDataStore()
	db := ccdb.NewLevelDBDataStore(i.storage_url_path)

	db.Inject(RootPrefix, commutative.NewPath())

	transitions := i.generateTransitions(db, accounts)
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

func (i *Initializer) loadGenesisAccounts() map[evmCommon.Address]*types.Account {
	file, err := os.OpenFile(i.accountFile, os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	accounts := make(map[evmCommon.Address]*types.Account)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}

		line = strings.TrimRight(line, "\n")
		segments := strings.Split(line, ",")
		balance, ok := new(big.Int).SetString(segments[2], 10)
		if !ok {
			panic(fmt.Sprintf("invalid balance in genesis accounts: %v", segments[2]))
		}

		accounts[evmCommon.HexToAddress(segments[1])] = &types.Account{
			Nonce:   0,
			Balance: balance,
		}
	}
	return accounts
}

func (i *Initializer) generateTransitions(db interfaces.Datastore, genesisAccounts map[evmCommon.Address]*types.Account) []interfaces.Univalue {
	batch := 10
	addresses := make([]evmCommon.Address, 0, batch)
	accounts := make([]*types.Account, 0, batch)
	index := 0
	transitions := make([]interfaces.Univalue, 0, len(genesisAccounts)*10)
	for addr, account := range genesisAccounts {
		if index%batch == 0 && index > 0 {
			transitions = append(transitions, getTransitions(db, addresses, accounts)...)
			addresses = make([]evmCommon.Address, 0, batch)
			accounts = make([]*types.Account, 0, batch)
		}
		addresses = append(addresses, addr)
		accounts = append(accounts, account)
		index++
	}
	if len(addresses) > 0 {
		transitions = append(transitions, getTransitions(db, addresses, accounts)...)
	}
	return transitions
}

func getTransitions(db interfaces.Datastore, addresses []evmCommon.Address, accounts []*types.Account) []interfaces.Univalue {
	url := ccurl.NewConcurrentUrl(db)
	api := ccapi.NewAPI(url)
	stateDB := eth.NewImplStateDB(api)
	stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for i, addr := range addresses {
		address := evmCommon.BytesToAddress(addr.Bytes())
		stateDB.CreateAccount(address)
		stateDB.SetBalance(address, accounts[i].Balance)
		stateDB.SetNonce(address, uint64(1))
	}
	_, transitions := url.ExportAll()

	return transitions
}
