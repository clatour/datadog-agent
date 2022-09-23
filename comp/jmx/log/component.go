// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build jmx
// +build jmx

// Package log implements a component to handle logging for JMX.
//
// This package configures its logfile using jmx.BundleParams#LogFile, but gets the
// remaining configuration parameters from core.BundleParams.
//
// At present, it configures and wraps the JMX logger in pkg/util/log, but will
// eventually be self-sufficient.
//
// The mock component does not read any configuration values, and redirects
// logging output to `t.Log(..)`, for ease of investigation when a test fails.
package log

import (
	"go.uber.org/fx"
)

// team: agent-shared-components

const componentName = "comp/jmx/log"

// Component is the component type.
type Component interface {
	// Info logs the given arguments, separated by spaces, at the info level
	Info(v ...interface{})

	// Error logs the given arguments, separated by spaces, at the error level,
	// and returns an error containing the messages.
	Error(v ...interface{}) error
}

// Mock is the mocked component type.
type Mock interface {
	Component

	// no further methods are defined.
}

// Module defines the fx options for this component.
var Module fx.Option = fx.Module(
	componentName,
	fx.Provide(newLogger),
)

// MockModule defines the fx options for the mock component.
var MockModule fx.Option = fx.Module(
	componentName,
	fx.Provide(newMockLogger),
)
