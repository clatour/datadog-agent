// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package health implements 'agent health'.
package health

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/DataDog/datadog-agent/cmd/agent/command"
	"github.com/DataDog/datadog-agent/cmd/agent/common"
	"github.com/DataDog/datadog-agent/pkg/api/util"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/status/health"
	"github.com/DataDog/datadog-agent/pkg/util/flavor"
)

// Commands returns a slice of subcommands for the 'agent' command.
func Commands(globalArgs *command.GlobalArgs) []*cobra.Command {
	cmd := &cobra.Command{
		Use:          "health",
		Short:        "Print the current agent health",
		Long:         ``,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if flavor.GetFlavor() == flavor.ClusterAgent {
				config.Datadog.SetConfigName("datadog-cluster")
			}

			// Set up config without secrets so that running the health command (e.g. from container
			// liveness probe script) does not trigger a secret backend command call.
			err := common.SetupConfigWithoutSecrets(globalArgs.ConfFilePath, "")
			if err != nil {
				return fmt.Errorf("unable to set up global agent configuration: %v", err)
			}

			err = config.SetupLogger(config.CoreLoggerName, config.GetEnvDefault("DD_LOG_LEVEL", "off"), "", "", false, true, false)
			if err != nil {
				fmt.Printf("Cannot setup logger, exiting: %v\n", err)
				return err
			}

			return requestHealth()
		},
	}
	return []*cobra.Command{cmd}
}
func requestHealth() error {
	c := util.GetClient(false) // FIX: get certificates right then make this true

	ipcAddress, err := config.GetIPCAddress()
	if err != nil {
		return err
	}

	var urlstr string
	if flavor.GetFlavor() == flavor.ClusterAgent {
		urlstr = fmt.Sprintf("https://%v:%v/status/health", ipcAddress, config.Datadog.GetInt("cluster_agent.cmd_port"))
	} else {
		urlstr = fmt.Sprintf("https://%v:%v/agent/status/health", ipcAddress, config.Datadog.GetInt("cmd_port"))
	}

	// Set session token
	err = util.SetAuthToken()
	if err != nil {
		return err
	}

	r, err := util.DoGet(c, urlstr, util.LeaveConnectionOpen)
	if err != nil {
		var errMap = make(map[string]string)
		json.Unmarshal(r, &errMap) //nolint:errcheck
		// If the error has been marshalled into a json object, check it and return it properly
		if e, found := errMap["error"]; found {
			err = fmt.Errorf(e)
		}

		fmt.Printf("Could not reach agent: %v \nMake sure the agent is running before requesting the status and contact support if you continue having issues. \n", err)
		return err
	}

	s := new(health.Status)
	if err = json.Unmarshal(r, s); err != nil {
		return fmt.Errorf("Error unmarshalling json: %s", err)
	}

	sort.Strings(s.Unhealthy)
	sort.Strings(s.Healthy)

	statusString := color.GreenString("PASS")
	if len(s.Unhealthy) > 0 {
		statusString = color.RedString("FAIL")
	}
	fmt.Fprintln(color.Output, fmt.Sprintf("Agent health: %s", statusString))

	if len(s.Healthy) > 0 {
		fmt.Fprintln(color.Output, fmt.Sprintf("=== %s healthy components ===", color.GreenString(strconv.Itoa(len(s.Healthy)))))
		fmt.Fprintln(color.Output, strings.Join(s.Healthy, ", "))
	}
	if len(s.Unhealthy) > 0 {
		fmt.Fprintln(color.Output, fmt.Sprintf("=== %s unhealthy components ===", color.RedString(strconv.Itoa(len(s.Unhealthy)))))
		fmt.Fprintln(color.Output, strings.Join(s.Unhealthy, ", "))
		return fmt.Errorf("found %d unhealthy components", len(s.Unhealthy))
	}

	return nil
}
