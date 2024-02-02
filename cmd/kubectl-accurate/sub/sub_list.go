package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

// newSubListCmd is an alias for the "kubectl-accurate list" command.
func newSubListCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	return newListCmd(streams, config)
}
