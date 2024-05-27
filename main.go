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
