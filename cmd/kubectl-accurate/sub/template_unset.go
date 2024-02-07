package sub

import (
	"context"
	"fmt"

	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type templateUnsetCmd struct {
	streams genericiooptions.IOStreams
	client  client.Client
	name    string
}

func newTemplateUnsetCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &templateUnsetCmd{}

	cmd := &cobra.Command{
		Use:   "unset NS",
		Short: "Unset template for NS namespace",
		Long:  `Unset template for NS namespace`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}

func (o *templateUnsetCmd) Fill(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	return nil
}

func (o *templateUnsetCmd) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	if _, ok := ns.Labels[constants.LabelTemplate]; !ok {
		return nil
	}

	delete(ns.Labels, constants.LabelTemplate)
	if err := o.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", o.name, err)
	}

	fmt.Fprintf(o.streams.Out, "unset template for %s\n", o.name)
	return nil
}
