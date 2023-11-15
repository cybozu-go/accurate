package sub

import (
	"context"
	"fmt"

	accuratev1 "github.com/cybozu-go/accurate/api/accurate/v1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type subCreateOpts struct {
	streams     genericclioptions.IOStreams
	client      client.Client
	name        string
	parent      string
	labels      map[string]string
	annotations map[string]string
}

func newSubCreateCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &subCreateOpts{}

	cmd := &cobra.Command{
		Use:   "create NAME NS",
		Short: "Create SubNamespace NAME in NS namespace",
		Long: `Create SubNamespace NAME in a namespace specified by NS.
This effectively creates a namespace named NAME as a sub-namespace of NS.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().StringToStringVar(&opts.labels, "labels", opts.labels, "the labels to be propagated to the sub-namespace. Example: a=b,c=d")
	cmd.Flags().StringToStringVar(&opts.annotations, "annotations", opts.annotations, "the annotations to be propagated to the sub-namespace. Example: a=b,c=d")
	return cmd
}

func (o *subCreateOpts) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
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

func (o *subCreateOpts) Run(ctx context.Context) error {
	ns := &corev1.Namespace{}
	err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns)
	if err == nil {
		return fmt.Errorf("namespace %s already exists", o.name)
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	sn := &accuratev1.SubNamespace{}
	sn.Namespace = o.parent
	sn.Name = o.name
	sn.Spec.Labels = o.labels
	sn.Spec.Annotations = o.annotations

	if err := o.client.Create(ctx, sn); err != nil {
		return fmt.Errorf("failed to create a SubNamespace: %w", err)
	}

	fmt.Fprintf(o.streams.Out, "SubNamespace %s is created in %s\n", o.name, o.parent)
	return nil
}
