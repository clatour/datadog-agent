// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build jmx
// +build jmx

package internal

// BundleParams defines the parameters for this bundle.
type BundleParams struct {
	// SeparateJmxLogFile determines whether the comp/jmx/log component should use
	// the `jmx_log_file` config to select its config file, or use the same file as
	// comp/core/log (derived from core.BundleParams).
	SeparateJmxLogFile bool
}
