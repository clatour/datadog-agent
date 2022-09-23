// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build jmx
// +build jmx

package run

import (
	"github.com/DataDog/datadog-agent/comp/jmx"
	"github.com/DataDog/datadog-agent/comp/jmx/log"
	"go.uber.org/fx"
)

// jmxOptions returns the JMX-related Fx options needed for the agent.
// These differ based on build flags.
func jmxOptions() fx.Option {
	return fx.Options(
		fx.Supply(jmx.BundleParams{
			SeparateJmxLogFile: true,
		}),
		jmx.Bundle,

		// Instantiate comp/jmx/log. Once the thing using this component requires
		// it, this will no longer be necessary.
		fx.Invoke(func(log.Component) {}),
	)
}
