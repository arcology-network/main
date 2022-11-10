package boot

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tmcmn "github.com/arcology-network/3rd-party/tm/common"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/component-lib/streamer"
	"github.com/arcology-network/main/config"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start monaco service Daemon",
	RunE:  startCmd,
}

func init() {

	flags := StartCmd.Flags()

	flags.String("global", "./config/global.json", "config file for global")
	flags.String("app", "./config/pool.json", "config file for application")
	flags.String("kafka", "./config/kafka.json", "config file for kafka")

}

func startCmd(cmd *cobra.Command, args []string) error {
	//clog.InitLog("consensus_com.log", viper.GetString("logcfg"), "consensus", viper.GetString("nname"), viper.GetInt("nidx"))

	globalConfigFile := viper.GetString("global") //os.Args[1]
	kafkaConfigFile := viper.GetString("kafka")   //os.Args[2]
	appConfigFile := viper.GetString("app")       // os.Args[3]

	globalConfig := config.LoadGlobalConfig(globalConfigFile)
	kafkaConfig := config.LoadKafkaConfig(kafkaConfigFile)
	appConfig := config.LoadAppConfig(appConfigFile)

	//viper.Set(cli.HomeFlag, os.ExpandEnv("$HOME/monacos/main"))

	initApp(globalConfig, kafkaConfig, appConfig)

	http.Handle("/streamer", promhttp.Handler())
	go http.ListenAndServe(appConfig.Settings.PrometheusListenAddr, nil)

	tmcmn.TrapSignal(func() {})

	return nil
}

func initApp(
	globalConfig config.GlobalConfig,
	kafkaConfig config.KafkaConfig,
	appConfig config.AppConfig,
) (*streamer.StatefulStreamer, []actor.IWorkerEx, []actor.IWorkerEx) {
	log.InitLog(
		appConfig.Settings.ServiceName+".log",
		globalConfig.LogConfigFile,
		appConfig.Settings.ServiceName,
		globalConfig.ClusterName,
		globalConfig.ClusterId,
	)

	broker := streamer.NewStatefulStreamer()
	workers := appConfig.InitApp(broker, globalConfig)
	downloaders, uploaders := kafkaConfig.InitKafka(broker, workers, globalConfig, appConfig)
	broker.Serve()

	for _, worker := range uploaders {
		worker.OnStart()
	}
	for _, worker := range workers {
		worker.OnStart()
	}
	for _, worker := range downloaders {
		worker.OnStart()
	}

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

	return broker, downloaders, uploaders
}
