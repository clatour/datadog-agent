// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build jmx
// +build jmx

package log

import (
	"testing"

	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/jmx/internal"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

func TestLogging(t *testing.T) {
	fxutil.Test(t, fx.Options(
		fx.Supply(core.BundleParams{}.LogForOneShot("TEST", "debug", false)),
		core.MockBundle,
		fx.Supply(internal.BundleParams{}),
		Module,
	), func(log Component) {
		log.Info("hello, world.")
	})
}
