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

package exec

import (
	eupk "github.com/arcology-network/eu"

	eucommon "github.com/arcology-network/eu/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/storage-committer/interfaces"
	stgcommitter "github.com/arcology-network/storage-committer/storage/committer"
	"github.com/arcology-network/streamer/actor"
)

type ExecutionImpl interface {
	Init(eu *eupk.EU, url *stgcommitter.StateCommitter)
	SetDB(db *interfaces.ReadOnlyStore)
	Exec(sequence *mtypes.ExecutingSequence, config *eucommon.Config, logger *actor.WorkerThreadLogger, gatherExeclog bool) (*exetyp.ExecutionResponse, error)
}
