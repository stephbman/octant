/*
 * Copyright (c) 2020 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"fmt"
	golog "log"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/vmware-tanzu/octant/internal/api"
	"github.com/vmware-tanzu/octant/internal/electron"
	"github.com/vmware-tanzu/octant/internal/log"
	"github.com/vmware-tanzu/octant/pkg/dash"
)

// Vars injected via ldflags by bundler
var (
	// AppName is the application name.
	AppName string
	// BuiltAt is when the application was built.
	BuiltAt string
	// VersionAstilectron is the astilectron version.
	VersionAstilectron string
	// VersionElectron is the electron version.
	VersionElectron string
)

func main() {
	z, err := log.Init(0, func(config zap.Config) zap.Config {
		config.ErrorOutputPaths = append(config.ErrorOutputPaths, "/tmp/octant.log")
		config.OutputPaths = append(config.OutputPaths, "/tmp/octant.log")

		return config
	})

	if err != nil {
		golog.Printf("unable to initialize logger: %v", err)
		os.Exit(1)
	}

	defer func() {
		_ = z.Sync()
	}()

	logger := log.Wrap(z.Sugar())
	ctx := log.WithLoggerContext(context.Background(), logger)

	if err := run(ctx); err != nil {
		logger.WithErr(err).Errorf("run")
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	logger := log.From(ctx)

	// Run bootstrap
	logger.With("built-at", BuiltAt).Infof("running app")

	options := electron.Options{
		AppName:            AppName,
		Asset:              Asset,
		AssetDir:           AssetDir,
		RestoreAssets:      RestoreAssets,
		VersionAstilectron: VersionAstilectron,
		VersionElectron:    VersionElectron,
	}

	e, err := electron.New(ctx, options)
	if err != nil {
		return fmt.Errorf("create electron app: %w", err)
	}

	viper.Set("disable-open-browser", true)
	viper.Set("proxy-frontend", "http://localhost:4200")

	// TODO: this port should be random.
	viper.Set(api.ListenerAddrKey, "127.0.0.1:7778")

	dashOptions := dash.Options{
		DisableClusterOverview: false,
		EnableOpenCensus:       false,
		KubeConfig:             clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename(),
		UserAgent:              fmt.Sprintf("octant-electron"), // TODO: create proper user agent
	}
	shutdownCh := make(chan bool, 1)
	startupCh := make(chan bool, 1)
	runCh := make(chan bool, 1)

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		if err := dash.Run(ctx, logger, shutdownCh, startupCh, dashOptions); err != nil {
			logger.WithErr(err).Errorf("dashboard failed")
			os.Exit(1)
		}
		runCh <- true
	}()

	<-startupCh
	if err := e.Start(ctx, "http://localhost:7778"); err != nil {
		return fmt.Errorf("start electron app: %w", err)
	}
	defer e.Stop()

	e.Wait()
	cancel()

	<-shutdownCh
	<-runCh

	return nil
}
