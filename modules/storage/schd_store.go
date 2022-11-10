package storage

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	"github.com/arcology-network/common-lib/transactional"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
)

var (
	schdStore     *SchdStore
	initSchdStore sync.Once
)

type SchdState struct {
	Height            uint64
	NewContracts      []ethcmn.Address
	ConflictionLefts  []ethcmn.Address
	ConflictionRights []ethcmn.Address
}

type SchdStore struct {
	actor.WorkerThread

	buf  *SchdState
	root string
	f    *os.File
}

func NewSchdStore(concurrency int, groupId string) actor.IWorkerEx {
	initSchdStore.Do(func() {
		schdStore = &SchdStore{}
		schdStore.Set(concurrency, groupId)
	})
	return schdStore
}

func (ss *SchdStore) Inputs() ([]string, bool) {
	return []string{actor.MsgBlockCompleted}, false
}

func (ss *SchdStore) Outputs() map[string]int {
	return map[string]int{}
}

func (ss *SchdStore) Config(params map[string]interface{}) {
	ss.root = params["root"].(string)
	if _, err := os.Stat(ss.root); os.IsNotExist(err) {
		os.Mkdir(ss.root, 0755)
	}

	var err error
	if ss.f, err = os.OpenFile(ss.root+"schd.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
		panic(err)
	}
}

func (ss *SchdStore) OnStart() {}

func (ss *SchdStore) OnMessageArrived(msgs []*actor.Message) error {
	if ss.buf != nil {
		ss.writeToFile(ss.buf)
	}
	return nil
}

func (ss *SchdStore) Save(ctx context.Context, state *SchdState, _ *int) error {
	ss.buf = state

	var na int
	return intf.Router.Call("transactionalstore", "AddData", &transactional.AddDataRequest{
		Data:        state,
		RecoverFunc: "schdstate",
	}, &na)
}

// DirectWrite only used in recover process.
func (ss *SchdStore) DirectWrite(ctx context.Context, state *SchdState, _ *int) error {
	return ss.writeToFile(state)
}

func (ss *SchdStore) Load(ctx context.Context, _ *int, states *[]SchdState) error {
	return ss.readFromFile(states)
}

func (ss *SchdStore) writeToFile(state *SchdState) error {
	str := formatState(state)
	if len(str) != 0 {
		if _, err := ss.f.WriteString(str); err != nil {
			return err
		}
	}
	return nil
}

func (ss *SchdStore) readFromFile(states *[]SchdState) error {
	f, err := os.Open(ss.root + "schd.txt")
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if line[len(line)-1] != '|' {
			panic("data corrupted")
		}

		segments := strings.Split(line[0:len(line)-1], "$")
		height, err := strconv.ParseInt(segments[0], 10, 64)
		if err != nil {
			panic(err)
		}
		*states = append(*states, SchdState{
			Height:            uint64(height),
			NewContracts:      parseAddressArray(segments[1]),
			ConflictionLefts:  parseAddressArray(segments[2]),
			ConflictionRights: parseAddressArray(segments[3]),
		})
	}
	return nil
}

func formatState(state *SchdState) string {
	if len(state.NewContracts) == 0 && len(state.ConflictionLefts) == 0 && len(state.ConflictionRights) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d$", state.Height))
	for i, addr := range state.NewContracts {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(addr.Hex())
	}
	sb.WriteString("$")
	for i, addr := range state.ConflictionLefts {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(addr.Hex())
	}
	sb.WriteString("$")
	for i, addr := range state.ConflictionRights {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(addr.Hex())
	}
	sb.WriteString("|\n")
	return sb.String()
}

func parseAddressArray(str string) []ethcmn.Address {
	if len(str) == 0 {
		return []ethcmn.Address{}
	}

	segments := strings.Split(str, ",")
	var addrs []ethcmn.Address
	for _, segment := range segments {
		addrs = append(addrs, ethcmn.HexToAddress(segment))
	}
	return addrs
}
