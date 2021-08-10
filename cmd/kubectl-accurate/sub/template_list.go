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

type templateListOpts struct {
	streams  genericclioptions.IOStreams
	client   client.Client
	template string
}

func newTemplateListCmd(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &templateListOpts{}

	cmd := &cobra.Command{
		Use:     "list [TEMPLATE]",
		Aliases: []string{"ls"},
		Short:   "List template namespace trees hierarchically",
		Long: `List template namespace trees hierarchically.
If TEMPLATE is not given, all template namespaces are shown hierarchically.
If TEMPLATE is given, only the tree under the TEMPLATE namespace will be shown.`,
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

func (o *templateListOpts) Fill(streams genericclioptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl

	if len(args) > 0 {
		o.template = args[0]
	}

	return nil
}

func (o *templateListOpts) Run(ctx context.Context) error {
	if o.template != "" {
		return o.showNS(ctx, o.template, 0)
	}

	templates := &corev1.NamespaceList{}
	if err := o.client.List(ctx, templates, client.MatchingLabels{constants.LabelType: constants.NSTypeTemplate}); err != nil {
		return fmt.Errorf("failed to list the template namespaces: %w", err)
	}

	for _, ns := range templates.Items {
		if err := o.showNS(ctx, ns.Name, 0); err != nil {
			return err
		}
	}
	return nil
}

func (o *templateListOpts) showNS(ctx context.Context, name string, level int) error {
	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", name, err)
	}

	subMark := " "
	if _, ok := ns.Labels[constants.LabelTemplate]; ok {
		subMark = "⮡"
	}
	fmt.Fprintf(o.streams.Out, "%*s%s%s\n", level, "", subMark, name)

	children := &corev1.NamespaceList{}
	if err := o.client.List(ctx, children, client.MatchingLabels{constants.LabelTemplate: name}); err != nil {
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
