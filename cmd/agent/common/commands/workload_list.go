package commands

import (
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-agent/cmd/agent/common"
	"github.com/DataDog/datadog-agent/pkg/api/util"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/flavor"
	"github.com/DataDog/datadog-agent/pkg/workloadmeta"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var verboseList bool

func WorkloadListCommand(loggerName config.LoggerName, confFilePath string, flagNoColor bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workload-list",
		Short: "Print the workload content of a running agent",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {

			if flagNoColor {
				color.NoColor = true
			}

			if flavor.GetFlavor() == flavor.ClusterAgent {
				config.Datadog.SetConfigName("datadog-cluster")
			}

			err := common.SetupConfigWithoutSecrets(confFilePath, "")
			if err != nil {
				return fmt.Errorf("unable to set up global agent configuration: %v", err)
			}

			err = config.SetupLogger(loggerName, config.GetEnvDefault("DD_LOG_LEVEL", "off"), "", "", false, true, false)
			if err != nil {
				fmt.Printf("Cannot setup logger, exiting: %v\n", err)
				return err
			}

			c := util.GetClient(false) // FIX: get certificates right then make this true

			// Set session token
			err = util.SetAuthToken()
			if err != nil {
				return err
			}
			ipcAddress, err := config.GetIPCAddress()
			if err != nil {
				return err
			}

			r, err := util.DoGet(c, workloadURL(verboseList, ipcAddress, config.Datadog.GetInt("cmd_port")), util.LeaveConnectionOpen)
			if err != nil {
				if r != nil && string(r) != "" {
					fmt.Fprintf(color.Output, "The agent ran into an error while getting the workload store information: %s\n", string(r))
				} else {
					fmt.Fprintf(color.Output, "Failed to query the agent (running?): %s\n", err)
				}
			}

			workload := workloadmeta.WorkloadDumpResponse{}
			err = json.Unmarshal(r, &workload)
			if err != nil {
				return err
			}

			workload.Write(color.Output)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verboseList, "verbose", "v", false, "print out a full dump of the workload store")

	return cmd

}

func workloadURL(verbose bool, address string, port int) string {
	var mode string
	if verbose {
		mode = "verbose"
	} else {
		mode = "short"
	}

	if flavor.GetFlavor() == flavor.ClusterAgent {
		return fmt.Sprintf("https://%v:%v/workload-list/%s", address, config.Datadog.GetInt("cluster_agent.cmd_port"), mode)
	} else {
		return fmt.Sprintf("https://%v:%v/agent/workload-list/%s", address, config.Datadog.GetInt("cmd_port"), mode)
	}
}
