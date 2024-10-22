package sub

import (
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	accuratev1 "github.com/cybozu-go/accurate/api/accurate/v1"
	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/controllers"
	"github.com/cybozu-go/accurate/hooks"
	"github.com/cybozu-go/accurate/pkg/config"
	"github.com/cybozu-go/accurate/pkg/indexing"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func subMain(ns, addr string, port int) error {
	logger := zap.New(zap.UseFlagOptions(&options.zapOpts))
	ctrl.SetLogger(logger)
	klog.SetLogger(logger)
	logger = ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add client-go objects: %w", err)
	}
	if err := accuratev1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Accurate v1 objects: %w", err)
	}
	if err := accuratev2alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Accurate v2alpha1 objects: %w", err)
	}
	if err := accuratev2.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add Accurate v2 objects: %w", err)
	}

	cfgData, err := os.ReadFile(options.configFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", options.configFile, err)
	}
	cfg := &config.Config{}
	if err := cfg.Load(cfgData); err != nil {
		return fmt.Errorf("unable to load the configuration file: %w", err)
	}

	restCfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}
	restCfg.QPS = float32(options.qps)
	restCfg.Burst = int(restCfg.QPS * 1.5)

	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				Unstructured: true,
			},
		},
		Metrics: server.Options{
			BindAddress: options.metricsAddr,
		},
		HealthProbeBindAddress:  options.probeAddr,
		LeaderElection:          true,
		LeaderElectionID:        options.leaderElectionID,
		LeaderElectionNamespace: ns,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    addr,
			Port:    port,
			CertDir: options.certDir,
		}),
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	ctx := ctrl.SetupSignalHandler()

	if err := cfg.Validate(mgr.GetRESTMapper()); err != nil {
		return fmt.Errorf("invalid configurations: %w", err)
	}
	if err := cfg.ValidateRBAC(ctx, mgr.GetClient(), mgr.GetRESTMapper()); err != nil {
		return fmt.Errorf("when validating RBAC to support configuration: %w", err)
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

	cloner := controllers.ResourceCloner{
		LabelKeyExcludes:      cfg.PropagateLabelKeyExcludes,
		AnnotationKeyExcludes: cfg.PropagateAnnotationKeyExcludes,
	}
	dec := admission.NewDecoder(scheme)

	// Namespace reconciler & webhook
	if err := indexing.SetupIndexForNamespace(ctx, mgr); err != nil {
		return fmt.Errorf("failed to setup indexer for namespaces: %w", err)
	}
	if err := (&controllers.NamespaceReconciler{
		Client:                     mgr.GetClient(),
		ResourceCloner:             cloner,
		LabelKeys:                  cfg.LabelKeys,
		AnnotationKeys:             cfg.AnnotationKeys,
		SubNamespaceLabelKeys:      cfg.SubNamespaceLabelKeys,
		SubNamespaceAnnotationKeys: cfg.SubNamespaceAnnotationKeys,
		Watched:                    watched,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create Namespace controller: %w", err)
	}
	hooks.SetupNamespaceWebhook(mgr, dec)

	// SubNamespace reconciler & webhook
	if err := indexing.SetupIndexForSubNamespace(ctx, mgr); err != nil {
		return fmt.Errorf("failed to setup indexer for subnamespaces: %w", err)
	}
	if err = (&controllers.SubNamespaceReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create SubNamespace controller: %w", err)
	}
	if err = hooks.SetupSubNamespaceWebhook(mgr, dec, cfg.NamingPolicyRegexps); err != nil {
		return fmt.Errorf("unable to create SubNamespace webhook: %w", err)
	}

	// Resource propagation controller
	for _, res := range watched {
		if err := indexing.SetupIndexForResource(ctx, mgr, res); err != nil {
			return fmt.Errorf("failed to setup indexer for %s: %w", res.GroupVersionKind().String(), err)
		}
		if err := controllers.NewPropagateController(res, cloner).SetupWithManager(mgr); err != nil {
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
