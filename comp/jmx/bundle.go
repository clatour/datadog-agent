// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build jmx
// +build jmx

// Package jmx implements the "jmx" bundle, providing components to support the
// interface between the agent and JMXFetch.
//
// This bundle depends on `comp/core`.
package jmx

import (
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/jmx/internal"
	"github.com/DataDog/datadog-agent/comp/jmx/log"
)

// team: agent-shared-components

const componentName = "comp/jmx"

// BundleParams defines the parameters for this bundle.
type BundleParams = internal.BundleParams

// Bundle defines the fx options for this bundle.
var Bundle = fx.Module(
	componentName,

	log.Module,
)

// MockBundle defines the mock fx options for this bundle.
var MockBundle = fx.Module(
	componentName,

	log.MockModule,
)
