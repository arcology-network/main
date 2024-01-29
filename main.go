package main

import (
	"os"

	"github.com/arcology-network/consensus-engine/cmd/tendermint/commands"
	"github.com/arcology-network/consensus-engine/libs/cli"
	"github.com/arcology-network/main/boot"
	"github.com/arcology-network/main/modules/consensus"
	"github.com/spf13/cobra"
)

func main() {

	var CoreCmd = &cobra.Command{
		Use:   "arcology",
		Short: "arcology main service",
		Long:  `arcology main service,It's the arcology logical structure as a conductor`,
	}

	CoreCmd.AddCommand(
		boot.StartCmd,
		consensus.InitCmd,
		commands.TestnetFilesCmd,
		consensus.MergeCmd,
	)

	cmd := cli.PrepareMainCmd(CoreCmd, "BC", os.ExpandEnv("$HOME/arcology/main"))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}

}
