// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver
// +build kubeapiserver

package app

import (
	"github.com/DataDog/datadog-agent/cmd/cluster-agent/commands"
)

func init() {
	ClusterAgentCmd.AddCommand(commands.Health(loggerName, &confPath, &flagNoColor))
}
