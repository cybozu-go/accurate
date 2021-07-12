package sub

import (
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	innuv1 "github.com/cybozu-go/innu/api/v1"
	"github.com/cybozu-go/innu/controllers"
	"github.com/cybozu-go/innu/hooks"
	"github.com/cybozu-go/innu/pkg/cluster"
	"github.com/cybozu-go/innu/pkg/config"
	"github.com/cybozu-go/innu/pkg/indexing"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func subMain(ns, addr string, port int) error {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&options.zapOpts)))
	logger := ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add client-go objects: %w", err)
	}
	if err := innuv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Innu objects: %w", err)
	}

	cfgData, err := os.ReadFile(options.configFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", options.configFile, err)
	}
	cfg := &config.Config{}
	if err := cfg.Load(cfgData); err != nil {
		return fmt.Errorf("unable to load the configuration file: %w", err)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		NewClient:               cluster.NewCachingClient,
		MetricsBindAddress:      options.metricsAddr,
		HealthProbeBindAddress:  options.probeAddr,
		LeaderElection:          true,
		LeaderElectionID:        options.leaderElectionID,
		LeaderElectionNamespace: ns,
		Host:                    addr,
		Port:                    port,
		CertDir:                 options.certDir,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	if err := cfg.Validate(mgr.GetRESTMapper()); err != nil {
		return fmt.Errorf("invalid configurations: %w", err)
	}

	watched := make([]*unstructured.Unstructured, len(cfg.Watches))
	for i := range cfg.Watches {
		gvk := &cfg.Watches[i]
		watched[i] = &unstructured.Unstructured{}
		watched[i].SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind,
		})
	}

	ctx := ctrl.SetupSignalHandler()
	dec, err := admission.NewDecoder(scheme)
	if err != nil {
		return fmt.Errorf("unable to create admission decoder: %w", err)
	}

	// Namespace reconciler & webhook
	if err := indexing.SetupIndexForNamespace(ctx, mgr); err != nil {
		return fmt.Errorf("failed to setup indexer for namespaces: %w", err)
	}
	if err := (&controllers.NamespaceReconciler{
		Client:         mgr.GetClient(),
		LabelKeys:      cfg.LabelKeys,
		AnnotationKeys: cfg.AnnotationKeys,
		Watched:        watched,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create Namespace controller: %w", err)
	}
	hooks.SetupNamespaceWebhook(mgr, dec)

	// SubNamespace reconciler & webhook
	if err = (&controllers.SubNamespaceReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create SubNamespace controller: %w", err)
	}
	hooks.SetupSubNamespaceWebhook(mgr, dec)

	// Resource propagation controller
	for _, res := range watched {
		if err := indexing.SetupIndexForResource(ctx, mgr, res); err != nil {
			return fmt.Errorf("failed to setup indexer for %s: %w", res.GroupVersionKind().String(), err)
		}
		if err := controllers.NewPropagateController(res).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create %s controller: %w", res.GroupVersionKind().String(), err)
		}
		logger.Info("watching", "gvk", res.GroupVersionKind().String())
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("problem running manager: %s", err)
	}
	return nil
}
