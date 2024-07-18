package sub

import (
	"context"
	"fmt"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type subCutOpts struct {
	streams genericiooptions.IOStreams
	client  client.Client
	name    string
}

func newSubCutCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &subCutOpts{}

	cmd := &cobra.Command{
		Use:   "cut NS",
		Short: "Make a sub-namespace NS a new root namespace",
		Long: `Make a sub-namespace NS a new root namespace.
The child sub-namespaces under NS will be moved along with it.`,
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

func (o *subCutOpts) Fill(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	return nil
}

func (o *subCutOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	parent, ok := ns.Labels[constants.LabelParent]
	if !ok {
		return fmt.Errorf("%s is not a sub-namespace", o.name)
	}

	delete(ns.Labels, constants.LabelParent)
	ns.Labels[constants.LabelType] = constants.NSTypeRoot
	if err := o.client.Update(ctx, ns); err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", o.name, err)
	}

	fmt.Fprintf(o.streams.Out, "cut %s as a root namespace\n", o.name)

	sn := &accuratev2.SubNamespace{}
	sn.Namespace = parent
	sn.Name = o.name

	if err := o.client.Delete(ctx, sn); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete SubNamespace %s/%s: %w", parent, o.name, err)
		}
	}

	return nil
}
