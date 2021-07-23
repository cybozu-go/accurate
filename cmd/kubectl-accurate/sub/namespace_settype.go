package sub

import (
	"context"
	"fmt"

	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type nsSetTypeOpts struct {
	streams genericclioptions.IOStreams
	client  client.Client
	name    string
	typ     string
}

func newNSSetTypeCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &nsSetTypeOpts{}

	cmd := &cobra.Command{
		Use:   "set-type NS TYPE",
		Short: "Set the type of a namespace",
		Long: `Set the type of a namespace NS.
Valid types are "root" or "template".

To unset the type, specify "none" as TYPE.`,
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

func (o *nsSetTypeOpts) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	o.typ = args[1]

	switch o.typ {
	case constants.NSTypeRoot, constants.NSTypeTemplate:
	case "none":
	default:
		return fmt.Errorf("invalid type: %s", o.typ)
	}
	return nil
}

func (o *nsSetTypeOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	current, ok := ns.Labels[constants.LabelType]
	if o.typ == "none" {
		if !ok {
			fmt.Fprintln(o.streams.Out, "nothing to do")
			return nil
		}
		delete(ns.Labels, constants.LabelType)
	} else {
		if current == o.typ {
			fmt.Fprintln(o.streams.Out, "nothing to do")
			return nil
		}
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels[constants.LabelType] = o.typ
	}

	if err := o.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", o.name, err)
	}

	fmt.Fprintln(o.streams.Out, "success")
	return nil
}
