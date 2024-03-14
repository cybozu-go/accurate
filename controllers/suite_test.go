package controllers

import (
	"context"
	"maps"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	accuratev1 "github.com/cybozu-go/accurate/api/accurate/v1"
	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/config"
	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/cybozu-go/accurate/pkg/feature"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var k8sCfg *rest.Config
var k8sClient client.Client
var scheme *runtime.Scheme
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(20 * time.Second)
	SetDefaultEventuallyPollingInterval(100 * time.Millisecond)
	SetDefaultConsistentlyDuration(5 * time.Second)
	SetDefaultConsistentlyPollingInterval(100 * time.Millisecond)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(
		zap.WriteTo(GinkgoWriter),
		zap.StacktraceLevel(zapcore.DPanicLevel),
		zap.Level(zapcore.Level(-5)),
	))

	// Some tests are still testing the propagate-generated feature
	Expect(config.DefaultMutableFeatureGate.SetFromMap(map[string]bool{string(feature.DisablePropagateGenerated): false})).To(Succeed())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	k8sCfg = cfg

	scheme = runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = accuratev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = accuratev2alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	komega.SetClient(k8sClient)

	// prepare resources
	ns := &corev1.Namespace{}
	ns.Name = "prop-root"
	ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	ns = &corev1.Namespace{}
	ns.Name = "prop-sub1"
	ns.Labels = map[string]string{constants.LabelParent: "prop-root"}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	ns = &corev1.Namespace{}
	ns.Name = "prop-sub2"
	ns.Labels = map[string]string{constants.LabelParent: "prop-root"}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	ns = &corev1.Namespace{}
	ns.Name = "prop-sub1-sub"
	ns.Labels = map[string]string{constants.LabelParent: "prop-sub1"}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	ns = &corev1.Namespace{}
	ns.Name = "prop-tmpl"
	ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	ns = &corev1.Namespace{}
	ns.Name = "prop-instance"
	ns.Labels = map[string]string{constants.LabelTemplate: "prop-tmpl"}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	// Create resources as they would look like before migration to SSA
	ns = &corev1.Namespace{}
	ns.Name = "pre-ssa-root"
	ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
	Expect(k8sClient.Create(context.Background(), ns)).To(Succeed())

	sn := &accuratev2alpha1.SubNamespace{}
	sn.Name = "pre-ssa-child"
	sn.Namespace = "pre-ssa-root"
	sn.Spec.Labels = map[string]string{
		"foo.glob/l": "glob",
		"bar.glob/l": "delete-me",
	}
	sn.Spec.Annotations = map[string]string{
		"foo.glob/a": "glob",
		"bar.glob/a": "delete-me",
	}
	Expect(k8sClient.Create(context.Background(), sn)).To(Succeed())

	ns = &corev1.Namespace{}
	ns.Name = sn.Name
	ns.Finalizers = []string{constants.Finalizer}
	ns.Labels = map[string]string{constants.LabelCreatedBy: constants.CreatedBy, constants.LabelParent: sn.Namespace}
	maps.Copy(ns.Labels, sn.Spec.Labels)
	ns.Annotations = sn.Spec.Annotations
	// Setting accurate-controller as field owner to simulate existing resource created by Accurate
	Expect(k8sClient.Create(context.Background(), ns, fieldOwner)).To(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
