// Copyright (c) straggler team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"os"
	"os/signal"
	"straggler/pkg/version"
	"syscall"

	"github.com/spf13/cobra"
)

var containerCMD = &cobra.Command{
	Use:   "container",
	Short: "stub for staggered pod containers",
	Run:   runContainer,
}

func init() {
	RootCMD.AddCommand(containerCMD)
}

func runContainer(command *cobra.Command, args []string) {
	logger := SetupTelemetryAndLogging()
	logger.Info("starting straggler container", "version", version.Build, "options", options)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	os.Exit(0)
}
