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

package ethapi

import (
	"context"
	"fmt"
	"net/http"

	jsonrpc "github.com/deliveroo/jsonrpc-go"
	"github.com/rs/cors"
)

var authmethods = map[string]jsonrpc.MethodFunc{
	"engine_forkchoiceUpdatedV2":               forkchoiceUpdatedV2,
	"engine_getPayloadV2":                      getPayloadV2,
	"engine_newPayloadV2":                      newPayloadV2,
	"engine_exchangeTransitionConfigurationV1": exchangeTransitionConfigurationV1,
}

func startAuthJsonRpc() {

	for k, v := range authmethods {
		methods[k] = v
	}

	server := jsonrpc.New()
	server.Use(func(next jsonrpc.Next) jsonrpc.Next {
		return func(ctx context.Context, params interface{}) (interface{}, error) {
			// method := jsonrpc.MethodFromContext(ctx)
			// fmt.Printf("***********************************************method: %v \t params:%v \n", method, params)
			return next(ctx, params)
		}
	})

	server.Register(methods)

	token, err := obtainJWTSecret(options.JwtFile)
	if err != nil {
		panic("read jwt secret file err!")
	}

	authSrv := newJWTHandler(token, server)

	c := cors.AllowAll()
	go http.ListenAndServe(fmt.Sprintf(":%d", options.AuthPort), c.Handler(authSrv))
}
