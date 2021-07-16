package sub

import (
	innuv1 "github.com/cybozu-go/innu/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func makeClient(config *genericclioptions.ConfigFlags) (client.Client, error) {
	cfg, err := config.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := innuv1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{Scheme: scheme})
}
