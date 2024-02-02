package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func newSubCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sub",
		Short: "sub-namespace command",
	}

	cmd.AddCommand(newSubCreateCmd(streams, config))
	cmd.AddCommand(newSubDeleteCmd(streams, config))
	cmd.AddCommand(newSubMoveCmd(streams, config))
	cmd.AddCommand(newSubGraftCmd(streams, config))
	cmd.AddCommand(newSubCutCmd(streams, config))
	cmd.AddCommand(newSubListCmd(streams, config))
	return cmd
}
