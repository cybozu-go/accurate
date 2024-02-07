package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func newNamespaceCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespace",
		Aliases: []string{"ns"},
		Short:   "namespace subcommand",
	}

	cmd.AddCommand(newNSDescribeCmd(streams, config))
	cmd.AddCommand(newNSSetTypeCmd(streams, config))
	return cmd
}
