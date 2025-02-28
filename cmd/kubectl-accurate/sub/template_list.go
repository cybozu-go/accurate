package sub

import (
	"context"
	"fmt"
	"sort"

	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type templateListOpts struct {
	streams  genericiooptions.IOStreams
	client   client.Client
	template string
}

func newTemplateListCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
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

func (o *templateListOpts) Fill(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
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
	allNamespaces := &corev1.NamespaceList{}
	if err := o.client.List(ctx, allNamespaces); err != nil {
		return fmt.Errorf("failed to list all namespaces: %w", err)
	}

	sort.Slice(allNamespaces.Items, func(i, j int) bool {
		return allNamespaces.Items[i].Name < allNamespaces.Items[j].Name
	})

	childMap := make(map[string][]*corev1.Namespace)
	nsMap := make(map[string]*corev1.Namespace)

	for i := range allNamespaces.Items {
		ns := &allNamespaces.Items[i]
		nsMap[ns.Name] = ns
		if parent, ok := ns.Labels[constants.LabelTemplate]; ok {
			childMap[parent] = append(childMap[parent], ns)
		}
	}

	var roots []*corev1.Namespace
	if o.template != "" {
		if rootNS, exists := nsMap[o.template]; exists {
			roots = append(roots, rootNS)
		} else {
			return fmt.Errorf("namespace %s not found. ensure the namespace exists and is correctly labeled", o.template)
		}
	} else {
		for _, ns := range allNamespaces.Items {
			if ns.Labels[constants.LabelType] == constants.NSTypeTemplate {
				if _, hasParent := ns.Labels[constants.LabelTemplate]; !hasParent {
					roots = append(roots, &ns)
				}
			}
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Name < roots[j].Name
	})

	for i, root := range roots {
		o.showNSRecursive(root, childMap, "", i == len(roots)-1)
	}

	return nil
}

func (o *templateListOpts) showNSRecursive(ns *corev1.Namespace, childMap map[string][]*corev1.Namespace, prefix string, isLast bool) {
	branch := "├── "
	if isLast {
		branch = "└── "
	}
	fmt.Fprintf(o.streams.Out, "%s%s%s\n", prefix, branch, ns.Name)

	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	children := childMap[ns.Name]
	numChildren := len(children)

	for i, child := range children {
		o.showNSRecursive(child, childMap, newPrefix, i == numChildren-1)
	}
}
