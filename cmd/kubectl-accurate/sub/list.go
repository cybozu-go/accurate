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

type listOptions struct {
	streams genericiooptions.IOStreams
	client  client.Client
	root    string
}

func newListCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
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

func (o *listOptions) Fill(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
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
	allNamespaces := &corev1.NamespaceList{}
	if err := o.client.List(ctx, allNamespaces); err != nil {
		return fmt.Errorf("failed to list all namespaces: %w", err)
	}

	sort.Slice(allNamespaces.Items, func(i, j int) bool {
		return allNamespaces.Items[i].Name < allNamespaces.Items[j].Name
	})

	nsMap := make(map[string]*corev1.Namespace)
	childMap := make(map[string][]*corev1.Namespace)

	for i := range allNamespaces.Items {
		ns := &allNamespaces.Items[i]
		nsMap[ns.Name] = ns
		if parent, ok := ns.Labels[constants.LabelParent]; ok {
			childMap[parent] = append(childMap[parent], ns)
		}
	}

	var roots []*corev1.Namespace
	if o.root != "" {
		if rootNS, exists := nsMap[o.root]; exists {
			roots = append(roots, rootNS)
		} else {
			return fmt.Errorf("namespace %s not found. ensure the namespace exists and is correctly labeled", o.root)
		}
	} else {
		for _, ns := range allNamespaces.Items {
			if ns.Labels[constants.LabelType] == constants.NSTypeRoot {
				roots = append(roots, &ns)
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

func (o *listOptions) showNSRecursive(ns *corev1.Namespace, childMap map[string][]*corev1.Namespace, prefix string, isLast bool) {
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
