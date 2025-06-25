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
	"context"
	"fmt"
	"strings"
	"sync"

	intf "github.com/arcology-network/streamer/interface"
)

type LockRequest struct {
	LockName string
	LockFrom string
}

const (
	locknames = "exec"
)

type modulesGuard struct {
	mlocks map[string]*sync.Mutex
}

func NewModulesGuard() *modulesGuard {
	mg := &modulesGuard{
		mlocks: map[string]*sync.Mutex{},
	}
	locks := strings.Split(locknames, ",")
	for i := range locks {
		mg.mlocks[locks[i]] = &sync.Mutex{}
		fmt.Printf("----main/components/storage/module_guard.go------lock init name:%v\n", locks[i])
	}
	return mg
}

// func (tg *modulesGuard) Config(params map[string]interface{}) {
// 	locknames := params["locknames"].(string)
// 	locks := strings.Split(locknames, ",")
// 	for i := range locks {
// 		tg.mlocks[locks[i]] = &sync.Mutex{}
// 		fmt.Printf("----main/components/storage/module_guard.go------lock init name:%v\n", locks[i])
// 	}
// }

func (tg *modulesGuard) Lock(ctx context.Context, request *LockRequest, locked *bool) error {
	fmt.Printf("-------components/storage/module_guard.go----Lock--name:%v---from:%v\n", request.LockName, request.LockFrom)
	tg.mlocks[request.LockName].Lock()
	return nil
}

func (tg *modulesGuard) UnLock(ctx context.Context, request *LockRequest, locked *bool) error {
	fmt.Printf("-------components/storage/module_guard.go----UnLock---name:%v--from:%v\n", request.LockName, request.LockFrom)
	tg.mlocks[request.LockName].Unlock()
	return nil
}

func RequestLock(name, from string) error {
	var response bool
	err := intf.Router.Call("global-lock", "Lock", &LockRequest{
		LockName: name,
		LockFrom: from,
	}, &response)
	if err != nil {
		return err
	}
	return nil
}

func ReleaseLock(name, from string) error {
	var response bool
	err := intf.Router.Call("global-lock", "UnLock", &LockRequest{
		LockName: name,
		LockFrom: from,
	}, &response)
	if err != nil {
		return err
	}
	return nil
}
