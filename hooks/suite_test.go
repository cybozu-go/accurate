package hooks

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	//+kubebuilder:scaffold:imports
	accuratev1 "github.com/cybozu-go/accurate/api/accurate/v1"
	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/config"
	"github.com/cybozu-go/accurate/pkg/indexing"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sClient client.Client
var testEnv *envtest.Environment
var cancelMgr context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel := context.WithCancel(context.TODO())
	cancelMgr = cancel

	scheme := runtime.NewScheme()
	err := accuratev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = accuratev2alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = admissionv1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		Scheme: scheme,
		CRDs:   loadCRDs(),
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "config", "webhook")},
		},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        server.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	err = indexing.SetupIndexForNamespace(ctx, mgr)
	Expect(err).NotTo(HaveOccurred())

	dec := admission.NewDecoder(scheme)
	SetupNamespaceWebhook(mgr, dec)

	conf := config.Config{
		NamingPolicies: []config.NamingPolicy{
			{
				Root:  "naming-policy-root-1",
				Match: "naming-policy-root-1-child",
			},
			{
				Root:  "naming-policy-root-2",
				Match: "naming-policy-root-2-child",
			},
			{
				Root:  ".+-match-.+",
				Match: ".+-match-.+",
			},
			{
				Root:  "^ns-root.+",
				Match: "^ns-root.+",
			},
			{
				Root:  "^app-(?P<team>.*)",
				Match: "^app-${team}-.*",
			},
			{
				Root:  "^app-(?P<team>[^-]*)-(?P<app>[^-]*)",
				Match: "^app-$team-$app-.*",
			},
			{
				Root:  "^unuse-naming-group-(?P<team>.*)",
				Match: "^unuse-naming-group-child1",
			},
		},
	}
	err = conf.Validate(mgr.GetRESTMapper())
	Expect(err).NotTo(HaveOccurred())
	err = SetupSubNamespaceWebhook(mgr, dec, conf.NamingPolicyRegexps)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		err = mgr.Start(ctx)
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())

})

var _ = AfterSuite(func() {
	cancelMgr()
	time.Sleep(50 * time.Millisecond)
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func loadCRDs() []*apiextensionsv1.CustomResourceDefinition {
	kOpts := krusty.MakeDefaultOptions()
	k := krusty.MakeKustomizer(kOpts)
	m, err := k.Run(filesys.FileSystemOrOnDisk{}, filepath.Join("..", "config", "crd"))
	Expect(err).To(Succeed())
	resources := m.Resources()

	crds := make([]*apiextensionsv1.CustomResourceDefinition, len(resources))
	for i := range resources {
		bytes, err := resources[i].MarshalJSON()
		Expect(err).To(Succeed())

		crd := &apiextensionsv1.CustomResourceDefinition{}
		err = json.Unmarshal(bytes, crd)
		Expect(err).To(Succeed())

		crds[i] = crd
	}

	return crds
}
