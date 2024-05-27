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

package modules

import (
	_ "github.com/arcology-network/main/modules/arbitrator"
	_ "github.com/arcology-network/main/modules/consensus"
	_ "github.com/arcology-network/main/modules/coordinator"
	_ "github.com/arcology-network/main/modules/core"
	_ "github.com/arcology-network/main/modules/eth-api"
	_ "github.com/arcology-network/main/modules/exec"
	_ "github.com/arcology-network/main/modules/gateway"
	_ "github.com/arcology-network/main/modules/p2p"
	_ "github.com/arcology-network/main/modules/pool"
	_ "github.com/arcology-network/main/modules/receipt-hashing"
	_ "github.com/arcology-network/main/modules/scheduler"
	_ "github.com/arcology-network/main/modules/state-sync"
	_ "github.com/arcology-network/main/modules/storage"
	_ "github.com/arcology-network/main/modules/toolkit"
	_ "github.com/arcology-network/main/modules/tpp"
	_ "github.com/arcology-network/main/modules/tx-sync"
)
