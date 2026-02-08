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

package boot

import (
	"net/http"

	"github.com/arcology-network/streamer/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/main/config"
	"github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/actor/rpc"
	brokerpk "github.com/arcology-network/streamer/broker"
	jetlib "github.com/arcology-network/streamer/jet/lib"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start arcology service Daemon",
	RunE:  startCmd,
}

func init() {

	flags := StartCmd.Flags()

	flags.String("global", "./config/global.yaml", "config file for global")
	flags.String("app", "./config/pool.yaml", "config file for application")
	flags.String("jet", "./config/jet.yaml", "config file for jet stream")
	flags.Bool("runAsL1", false, "run as l1 node")
}

func startCmd(cmd *cobra.Command, args []string) error {
	//clog.InitLog("consensus_com.log", viper.GetString("logcfg"), "consensus", viper.GetString("nname"), viper.GetInt("nidx"))

	globalConfigFile := viper.GetString("global") //os.Args[1]
	jetConfigFile := viper.GetString("jet")       //os.Args[2]
	appConfigFile := viper.GetString("app")       // os.Args[3]

	types.RunAsL1 = viper.GetBool("runAsL1")

	globalConfig, _ := config.LoadGlobalConfig(globalConfigFile)
	jetConfig, _ := jetlib.LoadConfig(jetConfigFile)
	appConfig, _ := config.LoadAppConfig(appConfigFile)

	initApp(globalConfig, jetConfig, appConfig)

	http.Handle("/streamer", promhttp.Handler())
	go http.ListenAndServe(appConfig.Settings.PrometheusListenAddr, nil)

	common.TrapSignal(func() {})

	return nil
}

func initApp(
	globalConfig *config.GlobalConfig,
	jetaConfig *jetlib.JetStreamConfig,
	appConfig *config.AppConfig,
) (*brokerpk.StatefulStreamer, map[string]actor.Business) {
	logger.InitLog(globalConfig.LogConfigFile, "")

	broker := brokerpk.NewStatefulStreamer()
	rpc.InitGlobalRPCFactory()
	rpc.InitGlobalRPCClient(broker, globalConfig.RpcConcurrent, globalConfig.RpcTimeoutSeconds)

	workers := appConfig.InitApp(broker, globalConfig, jetaConfig)

	broker.Serve()

	for _, worker := range workers {
		if _, ok := worker.(actor.Initializer); ok {
			msgs := worker.(actor.Initializer).InitMsgs()
			for _, msg := range msgs {
				broker.Send(msg.Name, msg)
			}
		}
	}

	for _, msg := range appConfig.StartMsgs {
		broker.Send(msg.Name, &msg)
	}

	return broker, workers
}
