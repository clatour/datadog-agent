// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build jmx
// +build jmx

package jmx

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/jmx/log"
)

func TestBundleDependencies(t *testing.T) {
	require.NoError(t, fx.ValidateApp(
		fx.Supply(core.BundleParams{}),
		core.Bundle,

		fx.Invoke(func(log.Component) {}),

		fx.Supply(BundleParams{}),
		Bundle))
}

func TestMockBundleDependencies(t *testing.T) {
	require.NoError(t, fx.ValidateApp(
		fx.Supply(fx.Annotate(t, fx.As(new(testing.TB)))),

		fx.Invoke(func(log.Component) {}),

		MockBundle))
}
