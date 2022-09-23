// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build jmx
// +build jmx

package log

import (
	"errors"
	"runtime"

	"github.com/DataDog/datadog-agent/cmd/agent/common"
	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/jmx/internal"
	pkgconfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// logger implements the component
type logger struct {
	// this component is currently implementing a thin wrapper around
	// pkg/util/log, and uses globals in that package.
}

func newLogger(coreParams core.BundleParams, params internal.BundleParams, config config.Component) (Component, error) {
	if coreParams.LogLevelFn == nil {
		return nil, errors.New("must call one of core.BundleParams.LogForOneShot or LogForDaemon")
	}
	var logFile string
	if params.SeparateJmxLogFile {
		// never log to file if disabled or on Android
		if !config.GetBool("disable_file_logging") && runtime.GOOS != "android" {
			logFile = config.GetString("jmx_log_file")
			if logFile == "" {
				logFile = common.DefaultJmxLogFile
			}
		}
	} else {
		// just do what the comp/core/log component did
		logFile = coreParams.LogFileFn(config)
	}
	err := pkgconfig.SetupJMXLogger(
		logFile,
		coreParams.LogSyslogURIFn(config),
		coreParams.LogSyslogRFCFn(config),
		coreParams.LogToConsoleFn(config),
		coreParams.LogFormatJSONFn(config))
	if err != nil {
		return nil, err
	}
	return &logger{}, nil
}

// Info implements Component#Info.
func (*logger) Info(v ...interface{}) { log.JMXInfo(v...) }

// Error implements Component#Error.
func (*logger) Error(v ...interface{}) error { return log.JMXError(v...) }
