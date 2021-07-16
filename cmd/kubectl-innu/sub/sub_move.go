package sub

import (
	"context"
	"fmt"

	innuv1 "github.com/cybozu-go/innu/api/v1"
	"github.com/cybozu-go/innu/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type subMoveOpts struct {
	streams genericclioptions.IOStreams
	client  client.Client
	name    string
	parent  string
	orphan  bool
}

func newSubMoveCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &subMoveOpts{}

	cmd := &cobra.Command{
		Use:   "move NS NEW_PARENT",
		Short: "Move a sub-namespace NS under the NEW_PARENT",
		Long: `Move a sub-namespace NS under the NEW_PARENT namespace.
NS must be a sub-namespace.
NEW_PARENT must be either a root or a sub-namespace.

A SubNamespace resource will be created in the NEW_PARENT namespace.
The SubNamespace in the original parent will be deleted if exists.

Use --leave-original to keep (ignore) the original SubNamespace.
In this case, the original SubNamespace will be marked as conflicted.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().BoolVar(&opts.orphan, "leave-original", false, "do not delete the SubNamespace in the original parent namespace")
	return cmd
}

func (o *subMoveOpts) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	o.parent = args[1]
	return nil
}

func (o *subMoveOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	orig, ok := ns.Labels[constants.LabelParent]
	if !ok {
		return fmt.Errorf("%s is not a sub-namespace", o.name)
	}

	if orig == o.parent {
		fmt.Fprintln(o.streams.Out, "parent is not changed")
		return nil
	}

	ns.Labels[constants.LabelParent] = o.parent
	if err := o.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", o.name, err)
	}

	fmt.Fprintf(o.streams.Out, "the parent has changed to %s\n", o.parent)

	if !o.orphan {
		oldSN := &innuv1.SubNamespace{}
		oldSN.Namespace = orig
		oldSN.Name = o.name
		err := o.client.Delete(ctx, oldSN)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete the original SubNamespace %s/%s: %w", orig, o.name, err)
			}
		} else {
			fmt.Fprintf(o.streams.Out, "deleted the original SubNamespace %s/%s\n", orig, o.name)
		}
	}

	sn := &innuv1.SubNamespace{}
	sn.Namespace = o.parent
	sn.Name = o.name
	if err := o.client.Create(ctx, sn); err != nil {
		return fmt.Errorf("failed to create SubNamespace in %s: %w", o.parent, err)
	}
	fmt.Fprintf(o.streams.Out, "created SubNamespace %s/%s\n", o.parent, o.name)
	return nil
}
