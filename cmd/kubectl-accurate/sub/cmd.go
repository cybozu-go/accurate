package sub

import (
	"os"

	"github.com/cybozu-go/accurate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewCmd creates the root *cobra.Command of `kubectl-accurate`.
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "accurate",
		Short:   "Subcommand for Accurate",
		Long:    `accurate is a subcommand of kubectl to manage Accurate features.`,
		Version: accurate.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return nil
		},
	}

	config := genericclioptions.NewConfigFlags(true)
	config.AddFlags(cmd.Flags())

	cmd.AddCommand(newListCmd(streams, config))
	cmd.AddCommand(newNamespaceCmd(streams, config))
	cmd.AddCommand(newTemplateCmd(streams, config))
	cmd.AddCommand(newSubCmd(streams, config))

	return cmd
}

// Execute executes `kubectl-accurate` command.
func Execute() {
	flags := pflag.NewFlagSet("kubectl-accurate", pflag.ExitOnError)
	pflag.CommandLine = flags

	cmd := NewCmd(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
