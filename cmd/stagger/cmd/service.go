// Copyright (c) stagger team and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package cmd

import (
	"os"
	"os/signal"
	"stagger/pkg/cmd"
	"stagger/pkg/version"
	"syscall"

	"github.com/spf13/cobra"
)

var serviceCMD = &cobra.Command{
	Use:   "service",
	Short: "manage admission controller and reconciler service",
	Run:   runService,
}

var (
	options = cmd.NewOptions()
)

func init() {
	EnrichCommand(serviceCMD, &options)
	RootCMD.AddCommand(serviceCMD)
}

func runService(command *cobra.Command, args []string) {
	logger := SetupTelemetryAndLogging()
	logger.Info("starting stagger service", "version", version.Build, "options", options)

	service, err := cmd.NewCMD(options, logger)
	if err != nil {
		logger.Info("failed to create command", "error", err)
		os.Exit(1)
	}
	if err := service.Start(logger); err != nil {
		logger.Info("failed to start service", "error", err)
		os.Exit(1)
	}

	logger.Info("startup sequence completed")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	if err := service.Stop(logger); err != nil {
		logger.Info("failed to stop service", "error", err)
		os.Exit(1)
	}
}
