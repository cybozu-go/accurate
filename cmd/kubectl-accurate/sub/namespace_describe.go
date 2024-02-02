package sub

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/cybozu-go/accurate/pkg/config"
	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type nsDescribeOpts struct {
	streams    genericiooptions.IOStreams
	client     client.Client
	name       string
	accurateNS string
}

func newNSDescribeCmd(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags) *cobra.Command {
	opts := &nsDescribeOpts{}
	cmd := &cobra.Command{
		Use:   "describe NS",
		Short: "Describe properties and propagated resources of NS namespace",
		Long:  `Describe properties and propagated resources of NS namespace.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Fill(streams, config, args); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&opts.accurateNS, "accurate-namespace", "accurate", "the namespace of accurate-controller")
	return cmd
}

func (o *nsDescribeOpts) Fill(streams genericiooptions.IOStreams, config *genericclioptions.ConfigFlags, args []string) error {
	o.streams = streams
	cl, err := makeClient(config)
	if err != nil {
		return err
	}
	o.client = cl
	o.name = args[0]
	return nil
}

func (o *nsDescribeOpts) printf(s string, args ...interface{}) {
	fmt.Fprintf(o.streams.Out, s, args...)
}

func (o *nsDescribeOpts) getChildren(ctx context.Context, k string) ([]string, error) {
	nsl := &corev1.NamespaceList{}
	if err := o.client.List(ctx, nsl, client.MatchingLabels{k: o.name}); err != nil {
		return nil, err
	}

	if len(nsl.Items) == 0 {
		return nil, nil
	}

	names := make([]string, len(nsl.Items))
	for i := range nsl.Items {
		names[i] = nsl.Items[i].Name
	}
	return names, nil
}

func (o *nsDescribeOpts) getConfig(ctx context.Context) (*config.Config, error) {
	deployment := &appsv1.Deployment{}
	if err := o.client.Get(ctx, client.ObjectKey{Namespace: o.accurateNS, Name: "accurate-controller-manager"}, deployment); err != nil {
		return nil, fmt.Errorf("failed to get deployment %s/%s: %w", o.accurateNS, "accurate-controller-manager", err)
	}

	var cmName string
	for _, vol := range deployment.Spec.Template.Spec.Volumes {
		if vol.Name != "config" {
			continue
		}
		if vol.ConfigMap == nil {
			return nil, fmt.Errorf("invalid config volume in Deployment %s/%s", o.accurateNS, "accurate-controller-manager")
		}
		cmName = vol.ConfigMap.Name
	}

	cm := &corev1.ConfigMap{}
	if err := o.client.Get(ctx, client.ObjectKey{Namespace: o.accurateNS, Name: cmName}, cm); err != nil {
		return nil, fmt.Errorf("failed to get configmap %s/%s: %w", o.accurateNS, cmName, err)
	}

	cfg := &config.Config{}
	if err := cfg.Load([]byte(cm.Data["config.yaml"])); err != nil {
		return nil, fmt.Errorf("failed to load config data: %w", err)
	}

	return cfg, nil
}

func (o *nsDescribeOpts) Run(ctx context.Context) error {
	cfg, err := o.getConfig(ctx)
	if err != nil {
		return err
	}

	ns := &corev1.Namespace{}
	if err := o.client.Get(ctx, client.ObjectKey{Name: o.name}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", o.name, err)
	}

	o.printf("Name: %s\n", ns.Name)

	typ := ns.Labels[constants.LabelType]
	switch typ {
	case constants.NSTypeRoot:
		o.printf("Type: %s\n", typ)
		children, err := o.getChildren(ctx, constants.LabelParent)
		if err != nil {
			return err
		}
		o.printf("# of children: %d\n", len(children))
	case constants.NSTypeTemplate:
		o.printf("Type: %s\n", typ)
		children, err := o.getChildren(ctx, constants.LabelTemplate)
		if err != nil {
			return err
		}
		o.printf("# of instances: %d\n", len(children))
	default:
		o.printf("Type: none\n")
	}

	if parent, ok := ns.Labels[constants.LabelParent]; ok {
		o.printf("Parent: %s\n", parent)
	}
	if tmpl, ok := ns.Labels[constants.LabelTemplate]; ok {
		o.printf("Template: %s\n", tmpl)
	}

	if len(cfg.Watches) == 0 {
		return nil
	}

	o.printf("\nResources:\n")
	w := tabwriter.NewWriter(o.streams.Out, 2, 8, 1, ' ', 0)
	fmt.Fprintln(w, "Kind\tName\tFrom\tMode")
	fmt.Fprintln(w, "--------\t--------\t--------\t--------")
	for _, gvk := range cfg.Watches {
		o.printResource(ctx, w, gvk)
	}
	w.Flush()
	return nil
}

func (o *nsDescribeOpts) printResource(ctx context.Context, w io.Writer, gvk metav1.GroupVersionKind) {
	objList := &unstructured.UnstructuredList{}
	objList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	})

	if err := o.client.List(ctx, objList, client.InNamespace(o.name)); err != nil {
		fmt.Fprintf(o.streams.ErrOut, "failed to list %s: %v\n", gvk.String(), err)
	}
	for _, obj := range objList.Items {
		anns := obj.GetAnnotations()
		from := anns[constants.AnnFrom]
		mode := anns[constants.AnnPropagate]
		if from == "" && mode == "" {
			continue
		}

		fmt.Fprintln(w, strings.Join([]string{gvk.Kind, obj.GetName(), from, mode}, "\t"))
	}
}
