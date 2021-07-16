package sub

import (
	"context"
	"fmt"

	"github.com/cybozu-go/innu/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type rootUnmarkOpts struct {
	streams genericclioptions.IOStreams
	client  client.Client
	name    string
}

func newRootUnmarkCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &rootUnmarkOpts{}

	cmd := &cobra.Command{
		Use:   "unmark NAMESPACE",
		Short: "Unmark a root NAMESPACE",
		Long: `Unmark a root NAMESPACE and make it an independent namespace.
This is done by removing "innu.cybozu.com/root=true" label from the namespace.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (o *rootUnmarkOpts) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	return nil
}

func (o *rootUnmarkOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	if ns.Labels[constants.LabelRoot] != "true" {
		fmt.Fprintln(o.streams.Out, "not a root namespace")
		return nil
	}

	delete(ns.Labels, constants.LabelRoot)
	if err := o.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", o.name, err)
	}

	fmt.Fprintf(o.streams.Out, "%s is no longer a root namespace\n", o.name)
	return nil
}
