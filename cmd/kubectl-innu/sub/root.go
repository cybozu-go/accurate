package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func newRootCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "root",
		Short: "root subcommand",
	}

	cmd.AddCommand(newRootMarkCmd(streams, config))
	cmd.AddCommand(newRootUnmarkCmd(streams, config))
	return cmd
}
