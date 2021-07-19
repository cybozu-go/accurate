package sub

import (
	"os"

	"github.com/cybozu-go/innu"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewCmd creates the root *cobra.Command of `kubectl-innu`.
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "innu",
		Short:   "Subcommand for Innu",
		Long:    `innu is a subcommand of kubectl to manage Innu features.`,
		Version: innu.Version,
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

// Execute executes `kubectl-innu` command.
func Execute() {
	flags := pflag.NewFlagSet("kubectl-innu", pflag.ExitOnError)
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
