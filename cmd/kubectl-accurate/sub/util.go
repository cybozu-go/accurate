package sub

import (
	accuratev1 "github.com/cybozu-go/accurate/api/accurate/v1"
	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func makeClient(config *genericclioptions.ConfigFlags) (client.Client, error) {
	cfg, err := config.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := accuratev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := accuratev2alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{Scheme: scheme})
}
