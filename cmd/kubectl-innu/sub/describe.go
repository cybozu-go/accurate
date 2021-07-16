package sub

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func newDescribeCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use: "describe NAMESPACE",
	}
	return cmd
}
