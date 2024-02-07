package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func newTemplateCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "template subcommand",
	}

	cmd.AddCommand(newTemplateListCmd(streams, config))
	cmd.AddCommand(newTemplateSetCmd(streams, config))
	cmd.AddCommand(newTemplateUnsetCmd(streams, config))
	return cmd
}
