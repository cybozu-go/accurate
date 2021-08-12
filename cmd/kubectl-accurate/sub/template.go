package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func newTemplateCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "template subcommand",
	}

	cmd.AddCommand(newTemplateListCmd(streams, config))
	cmd.AddCommand(newTemplateSetCmd(streams, config))
	cmd.AddCommand(newTemplateUnsetCmd(streams, config))
	return cmd
}
