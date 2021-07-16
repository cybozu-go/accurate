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

type templateSetOpts struct {
	streams  genericclioptions.IOStreams
	client   client.Client
	template string
	name     string
}

func newTemplateSetCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &templateSetOpts{}

	cmd := &cobra.Command{
		Use:   "set TEMPLATE NS",
		Short: "Set TEMPLATE as the template of NS namespace",
		Long: `Set a template namespace for a namespace NS.
TEMPLATE and NS are namespace names.
NS must be a root or an independent namespace.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (o *templateSetOpts) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.template = args[0]
	o.name = args[1]
	return nil
}

func (o *templateSetOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels[constants.LabelTemplate] = o.template
	if err := o.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", o.name, err)
	}

	fmt.Fprintf(o.streams.Out, "set %s as a template of %s\n", o.template, o.name)
	return nil
}
