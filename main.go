package main

import (
	"os"

	"github.com/HPISTechnologies/3rd-party/tm/cli"
	"github.com/HPISTechnologies/consensus-engine/cmd/tendermint/commands"
	"github.com/HPISTechnologies/main/boot"
	"github.com/HPISTechnologies/main/modules/consensus"
	"github.com/spf13/cobra"
)

func main() {

	var CoreCmd = &cobra.Command{
		Use:   "monaco",
		Short: "monaco main service",
		Long:  `monaco main service,It's the Monaco logical structure as a conductor`,
	}

	CoreCmd.AddCommand(
		boot.StartCmd,
		consensus.InitCmd,
		commands.TestnetFilesCmd,
		consensus.MergeCmd,
	)

	cmd := cli.PrepareMainCmd(CoreCmd, "BC", os.ExpandEnv("$HOME/monacos/main"))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}

}
