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

type rootMarkOpts struct {
	streams genericclioptions.IOStreams
	client  client.Client
	name    string
}

func newRootMarkCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &rootMarkOpts{}

	cmd := &cobra.Command{
		Use:   "mark NAMESPACE",
		Short: "Mark NAMESPACE as a root namespace",
		Long: `Mark an independent namespace as a root namespace.
This is done by labelling the namespace with "innu.cybozu.com/root=true".`,
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

func (o *rootMarkOpts) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	return nil
}

func (o *rootMarkOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	if ns.Labels[constants.LabelRoot] == "true" {
		fmt.Fprintln(o.streams.Out, "marked already")
		return nil
	}

	if ns.Labels == nil {
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
	} else {
		ns.Labels[constants.LabelRoot] = "true"
	}

	if err := o.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", o.name, err)
	}

	fmt.Fprintln(o.streams.Out, "marked as a root namespace")
	return nil
}
