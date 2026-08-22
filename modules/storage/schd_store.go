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

package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arcology-network/common-lib/storage/transactional"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

// var (
// 	schdStore *SchdStore
// )

type SchdStore struct {
	// buf  *mtypes.SchdState
	root string
	f    *os.File
}

func NewSchdStore() actor.Business {
	return &SchdStore{}
}

func (ss *SchdStore) Inputs() ([]string, bool) {
	return []string{}, false
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
	if ss.f, err = os.OpenFile(filepath.Join(ss.root, "schd.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
		panic(err)
	}
}

func (ss *SchdStore) RpcConfig() (string, int) {
	return "schdstore", 20
}

func (ss *SchdStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Save", ss.Save)
	reg.Register("SaveBack", ss.SaveBack)
	reg.Register("DirectWrite", ss.DirectWrite)
	reg.Register("Load", ss.Load)
}

func (ss *SchdStore) Save(ctx *actor.ActionContext) error {
	stat := ctx.RPC.Request.(*mtypes.SchdState)
	if stat == nil {
		ctx.ExecCtx.SendRpcResponse("", "")
		return nil
	}
	ss.writeToFile(stat)

	// ctx.ExecCtx.StartCasecade()
	ctx.ExecCtx.InvokeRPC("transactionalstore", "AddData", &transactional.AddDataRequest{
		Data:        stat,
		RecoverFunc: "schdstate",
	}, "SaveBack")
	return nil
}

func (ss *SchdStore) SaveBack(ctx *actor.ActionContext) error {
	// ctx.ExecCtx.EndCasecade()
	ctx.ExecCtx.SendRpcResponse("", "")
	return nil
}

func (ss *SchdStore) DirectWrite(ctx *actor.ActionContext) error {
	stat := ctx.RPC.Request.(*mtypes.SchdState)
	ss.writeToFile(stat)
	ctx.ExecCtx.SendRpcResponse("", "")
	return nil
}

func (ss *SchdStore) Load(ctx *actor.ActionContext) error {
	states := &[]mtypes.SchdState{}
	ss.readFromFile(states)
	ctx.ExecCtx.SendRpcResponse("", *states)
	return nil
}

func (ss *SchdStore) writeToFile(state *mtypes.SchdState) error {
	str := formatState(state)
	if len(str) != 0 {
		if _, err := ss.f.WriteString(str); err != nil {
			return err
		}
	}
	return nil
}

func (ss *SchdStore) readFromFile(states *[]mtypes.SchdState) error {
	f, err := os.Open(filepath.Join(ss.root, "schd.txt"))
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
		*states = append(*states, mtypes.SchdState{
			Height:                uint64(height),
			NewContracts:          parseAddressArray(segments[1]),
			ConflictionLefts:      parseAddressArray(segments[2]),
			ConflictionRights:     parseAddressArray(segments[3]),
			ConflictionLeftSigns:  parseSignArray(segments[4]),
			ConflictionRightSigns: parseSignArray(segments[5]),
		})
	}
	return nil
}

func formatState(state *mtypes.SchdState) string {
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
	sb.WriteString("$")
	for i, sign := range state.ConflictionLeftSigns {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%x", sign))
	}
	sb.WriteString("$")
	for i, sign := range state.ConflictionRightSigns {
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%x", sign))
	}
	sb.WriteString("|\n")
	return sb.String()
}

func parseAddressArray(str string) []evmCommon.Address {
	if len(str) == 0 {
		return []evmCommon.Address{}
	}

	segments := strings.Split(str, ",")
	var addrs []evmCommon.Address
	for _, segment := range segments {
		addrs = append(addrs, evmCommon.HexToAddress(segment))
	}
	return addrs
}

func parseSignArray(str string) [][4]byte {
	if len(str) == 0 {
		return [][4]byte{}
	}

	segments := strings.Split(str, ",")
	var signs [][4]byte
	for _, segment := range segments {
		signs = append(signs, [4]byte([]byte(segment)))
	}
	return signs
}
