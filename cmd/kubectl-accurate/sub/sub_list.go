package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// newSubListCmd is an alias for the "kubectl-accurate list" command.
func newSubListCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	return newListCmd(streams, config)
}
