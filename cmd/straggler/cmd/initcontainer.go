// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var initContainerCMD = &cobra.Command{
	Use:   "initcontainer",
	Short: "stub for staggered pod init containers",
	Run:   runInitContainer,
}

func init() {
	RootCMD.AddCommand(initContainerCMD)
}

func runInitContainer(command *cobra.Command, args []string) {
	os.Exit(0)
}
