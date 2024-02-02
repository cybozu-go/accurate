package sub

import (
	"context"
	"fmt"

	accuratev1 "github.com/cybozu-go/accurate/api/accurate/v1"
	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type subDeleteOpts struct {
	streams genericiooptions.IOStreams
	client  client.Client
	name    string
}

func newSubDeleteCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &subDeleteOpts{}

	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a SubNamespace NAME",
		Long: `Delete a SubNamespace NAME in the parent namespace of NAME.
This effectively deletes the namespace NAME.`,
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

func (o *subDeleteOpts) Fill(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	return nil
}

func (o *subDeleteOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	parent, ok := ns.Labels[constants.LabelParent]
	if !ok {
		return fmt.Errorf("namespace %s is not a sub-namespace", o.name)
	}

	sn := &accuratev1.SubNamespace{}
	sn.Namespace = parent
	sn.Name = o.name
	if err := o.client.Delete(ctx, sn); err != nil {
		return fmt.Errorf("failed to delete SubNamespace %s/%s: %w", parent, o.name, err)
	}

	fmt.Fprintf(o.streams.Out, "deleted SubNamespace %s/%s\n", parent, o.name)
	return nil
}
