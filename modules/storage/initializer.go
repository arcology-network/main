package storage

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"strings"

	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	cstore "github.com/HPISTechnologies/common-lib/cachedstorage"
	cmntyp "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	ccurl "github.com/HPISTechnologies/concurrenturl/v2"
	urlcmn "github.com/HPISTechnologies/concurrenturl/v2/common"
	urltyp "github.com/HPISTechnologies/concurrenturl/v2/type"
	"github.com/HPISTechnologies/concurrenturl/v2/type/commutative"
	evmcmn "github.com/HPISTechnologies/evm/common"
	stypes "github.com/HPISTechnologies/main/modules/storage/types"
	adaptor "github.com/HPISTechnologies/vm-adaptor/evm"
)

type Initializer struct {
	actor.WorkerThread

	accountFile        string
	storage_url_path   string
	storage_url_shards int
	storage_url_depts  int
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
	i.storage_url_shards = int(params["storage_url_shards"].(float64))
	i.storage_url_depts = int(params["storage_url_depts"].(float64))
}

func (i *Initializer) InitMsgs() []*actor.Message {
	// TODO: load latest block height from db.
	height := 0

	var db urlcmn.DatastoreInterface
	var rootHash ethcmn.Hash
	if height == 0 {
		db, rootHash = i.initGenesisAccounts()
		var na int
		intf.Router.Call("statestore", "Save", &State{
			Height:     0,
			ParentHash: ethcmn.Hash{},
			ParentRoot: rootHash,
		}, &na)
	} else {
		panic("not yet implemented")
	}

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
	var na int
	// filedb, err := cstore.NewFileDB(i.storage_url_path, uint32(i.storage_url_shards), uint8(i.storage_url_depts))
	// if err != nil {
	// 	panic("create filedb err!:" + err.Error())
	// }
	db := cstore.NewDataStore(
		nil,
		cstore.NewCachePolicy(math.MaxUint64, 1),
		//filedb,
		stypes.NewMemoryDB(),
		func(v interface{}) []byte { return urltyp.ToBytes(v) },
		func(bytes []byte) interface{} { return urltyp.FromBytes(bytes) },
	)

	intf.Router.Call("urlstore", "Init", db, &na)

	platform := urlcmn.NewPlatform()
	meta, _ := commutative.NewMeta(platform.Eth10Account())
	db.Inject(platform.Eth10Account(), meta)

	transitions := i.generateTransitions(db, accounts)
	url := ccurl.NewConcurrentUrl(db)
	url.Indexer().Import(transitions)
	url.PostImport()
	url.Precommit([]uint32{0})
	keys, values := url.Indexer().KVs()
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
			Nonce:   1,
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
		stateDB.SetNonce(address, accounts[i].Nonce)
	}
	_, transitions := url.Export(false)
	return transitions
}
