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

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cstore "github.com/arcology-network/common-lib/cachedstorage"
	"github.com/arcology-network/common-lib/transactional"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/storage"
	ccurl "github.com/arcology-network/concurrenturl/v2"
	urlcmn "github.com/arcology-network/concurrenturl/v2/common"
	urltyp "github.com/arcology-network/concurrenturl/v2/type"
	"github.com/arcology-network/concurrenturl/v2/type/commutative"
	"github.com/arcology-network/consensus-engine/state"
	evmcmn "github.com/arcology-network/evm/common"
	"github.com/arcology-network/main/modules/core"
	adaptor "github.com/arcology-network/vm-adaptor/evm"
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

	var db urlcmn.DatastoreInterface
	var rootHash ethcmn.Hash
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
		var na int
		intf.Router.Call("statestore", "Save", &State{
			Height:     0,
			ParentHash: ethcmn.Hash{},
			ParentRoot: rootHash,
		}, &na)
		parentinfo := &cmntyp.ParentInfo{
			ParentHash: ethcmn.Hash{},
			ParentRoot: rootHash,
		}
		_, block, err := core.CreateBlock(parentinfo, uint64(0), big.NewInt(0), ethcmn.Address{}, rootHash, uint64(0), ethcmn.Hash{}, ethcmn.Hash{}, [][]byte{})
		if err != nil {
			panic("Create genesis block err!")
		}
		intf.Router.Call("blockstore", "Save", block, &na)
	} else {
		db = cstore.NewDataStore(
			nil,
			cstore.NewCachePolicy(cstore.Cache_Quota_Full, 1),
			cstore.NewParaBadgerDB(i.storage_url_path, urlcmn.Eth10AccountShard),
			func(v interface{}) []byte { return urltyp.ToBytes(v) },
			func(bytes []byte) interface{} { return urltyp.FromBytes(bytes) },
		)

		platform := urlcmn.NewPlatform()
		meta, _ := commutative.NewMeta(platform.Eth10Account())
		db.Inject(platform.Eth10Account(), meta)

		// Register recover function.
		transactional.RegisterRecoverFunc("urlupdate", func(_ interface{}, bs []byte) error {
			var updates storage.UrlUpdate
			if err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&updates); err != nil {
				fmt.Printf("Error decoding UrlUpdate, err = %v\n", err)
				return err
			}

			values := make([]interface{}, len(updates.EncodedValues))
			for i, v := range updates.EncodedValues {
				values[i] = urltyp.FromBytes(v)
			}
			db.BatchInject(updates.Keys, values)
			fmt.Printf("[storage.Initializer] Recover urlupdate.\n")
			return nil
		})
		transactional.RegisterRecoverFunc("parentinfo", func(_ interface{}, bs []byte) error {
			var pi cmntyp.ParentInfo
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

func (i *Initializer) initGenesisAccounts() (urlcmn.DatastoreInterface, ethcmn.Hash) {
	accounts := i.loadGenesisAccounts()
	// filedb, err := cstore.NewFileDB(i.storage_url_path, uint32(i.storage_url_shards), uint8(i.storage_url_depts))
	// if err != nil {
	// 	panic("create filedb err!:" + err.Error())
	// }
	db := cstore.NewDataStore(
		nil,
		cstore.NewCachePolicy(cstore.Cache_Quota_Full, 1),
		// stypes.NewMemoryDB(),
		cstore.NewParaBadgerDB(i.storage_url_path, urlcmn.Eth10AccountShard),
		func(v interface{}) []byte { return urltyp.ToBytes(v) },
		func(bytes []byte) interface{} { return urltyp.FromBytes(bytes) },
	)

	platform := urlcmn.NewPlatform()
	meta, _ := commutative.NewMeta(platform.Eth10Account())
	db.Inject(platform.Eth10Account(), meta)

	transitions := i.generateTransitions(db, accounts)
	url := ccurl.NewConcurrentUrl(db)
	url.Import(transitions)
	url.PostImport()
	url.Precommit([]uint32{0})
	keys, values := url.KVs()
	encodedValues := make([][]byte, 0, len(values))
	for _, v := range values {
		if v != nil {
			univalue := v.(urlcmn.UnivalueInterface)
			data := urltyp.ToBytes(univalue.Value())
			encodedValues = append(encodedValues, data)
		} else {
			encodedValues = append(encodedValues, []byte{})
		}
	}

	merkle := urltyp.NewAccountMerkle(urlcmn.NewPlatform())
	merkle.Import(transitions)
	rootHash := calcRootHash(merkle, ethcmn.Hash{}, keys, encodedValues)
	url.Postcommit()
	url.SaveToDB()
	return db, rootHash
}

func (i *Initializer) loadGenesisAccounts() map[ethcmn.Address]*cmntyp.Account {
	file, err := os.OpenFile(i.accountFile, os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	accounts := make(map[ethcmn.Address]*cmntyp.Account)
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

		accounts[ethcmn.HexToAddress(segments[1])] = &cmntyp.Account{
			Nonce:   0,
			Balance: balance,
		}
	}
	return accounts
}

func (i *Initializer) generateTransitions(db urlcmn.DatastoreInterface, genesisAccounts map[ethcmn.Address]*cmntyp.Account) []urlcmn.UnivalueInterface {
	batch := 10
	addresses := make([]ethcmn.Address, 0, batch)
	accounts := make([]*cmntyp.Account, 0, batch)
	index := 0
	transitions := make([]urlcmn.UnivalueInterface, 0, len(genesisAccounts)*10)
	for addr, account := range genesisAccounts {
		if index%batch == 0 && index > 0 {
			transitions = append(transitions, getTransitions(db, addresses, accounts)...)
			addresses = make([]ethcmn.Address, 0, batch)
			accounts = make([]*cmntyp.Account, 0, batch)
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

func getTransitions(db urlcmn.DatastoreInterface, addresses []ethcmn.Address, accounts []*cmntyp.Account) []urlcmn.UnivalueInterface {
	url := ccurl.NewConcurrentUrl(db)
	stateDB := adaptor.NewStateDBV2(nil, db, url)
	stateDB.Prepare(evmcmn.Hash{}, evmcmn.Hash{}, 0)
	for i, addr := range addresses {
		address := evmcmn.BytesToAddress(addr.Bytes())
		stateDB.CreateAccount(address)
		stateDB.SetBalance(address, accounts[i].Balance)
		//stateDB.SetNonce(address, accounts[i].Nonce)
	}
	_, transitions := url.Export(false)
	return transitions
}
