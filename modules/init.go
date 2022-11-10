package modules

import (
	_ "github.com/arcology-network/main/modules/arbitrator"
	_ "github.com/arcology-network/main/modules/consensus"
	_ "github.com/arcology-network/main/modules/coordinator"
	_ "github.com/arcology-network/main/modules/core"

	// _ "github.com/arcology-network/main/modules/core/v2"
	_ "github.com/arcology-network/main/modules/eth-api"
	_ "github.com/arcology-network/main/modules/exec/v2"
	_ "github.com/arcology-network/main/modules/gateway"
	_ "github.com/arcology-network/main/modules/p2p"
	_ "github.com/arcology-network/main/modules/pool"
	_ "github.com/arcology-network/main/modules/receipt-hashing"
	_ "github.com/arcology-network/main/modules/scheduler/v2"
	_ "github.com/arcology-network/main/modules/state-sync"
	_ "github.com/arcology-network/main/modules/storage"
	_ "github.com/arcology-network/main/modules/toolkit"
	_ "github.com/arcology-network/main/modules/tpp"
	_ "github.com/arcology-network/main/modules/tx-sync"
)
