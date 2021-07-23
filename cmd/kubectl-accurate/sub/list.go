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

const indent = 4

type listOptions struct {
	streams genericclioptions.IOStreams
	client  client.Client
	root    string
}

func newListCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:     "list [ROOT]",
		Aliases: []string{"ls"},
		Short:   "List namespace trees hierarchically",
		Long: `List namespace trees hierarchically.
If ROOT is not given, all root namespaces and their children will be shown.
If ROOT is given, only the tree under the ROOT namespace will be shown.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}
	return cmd
}

func (o *listOptions) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl

	if len(args) > 0 {
		o.root = args[0]
	}

	return nil
}

func (o *listOptions) Run(ctx context.Context) error {
	if o.root != "" {
		return o.showNS(ctx, o.root, 0)
	}

	roots := &corev1.NamespaceList{}
	if err := o.client.List(ctx, roots, client.MatchingLabels{constants.LabelType: constants.NSTypeRoot}); err != nil {
		return fmt.Errorf("failed to list the root namespaces: %w", err)
	}

	for _, ns := range roots.Items {
		if err := o.showNS(ctx, ns.Name, 0); err != nil {
			return err
		}
	}
	return nil
}

func (o *listOptions) showNS(ctx context.Context, name string, level int) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", name, err)
	}

	subMark := " "
	if _, ok := ns.Labels[constants.LabelParent]; ok {
		subMark = "⮡"
	}
	fmt.Fprintf(o.streams.Out, "%*s%s%s\n", level, "", subMark, name)

	children := &corev1.NamespaceList{}
	if err := o.client.List(ctx, children, client.MatchingLabels{constants.LabelParent: name}); err != nil {
		return fmt.Errorf("failed to list the children of %s: %w", name, err)
	}

	level += indent
	for _, child := range children.Items {
		if err := o.showNS(ctx, child.Name, level); err != nil {
			return err
		}
	}
	return nil
}
