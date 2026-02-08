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

package businessCase

import (
	cstorage "github.com/arcology-network/main/components/storage"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
)

func init() {

	actor.Factory.Register("general_url_async", func() actor.Business {
		return cstorage.NewDBHandlerAsync(scommon.MsgGeneralDB, scommon.MsgGeneralPrecommit, scommon.MsgGeneralCommit, scommon.MsgGeneralCompleted)
	})

	actor.Factory.Register("nonce_url_async", func() actor.Business {
		return cstorage.NewDBHandlerAsync(scommon.MsgNonceDB, scommon.MsgNoncePrecommit, scommon.MsgNonceCommit, scommon.MsgNonceCompleted)
	})

}
