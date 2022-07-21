package e2e

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os"

	accuratev1 "github.com/cybozu-go/accurate/api/v1"
	"github.com/cybozu-go/accurate/pkg/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

//go:embed testdata/role.yaml
var roleYAML []byte

//go:embed testdata/resourceQuota.yaml
var resourceQuota []byte

//go:embed testdata/serviceaccount.yaml
var serviceAccountYAML []byte

var sealedJSON []byte

func init() {
	data, err := os.ReadFile("sealed.json")
	if err != nil {
		panic(err)
	}
	sealedJSON = data
}

var _ = Describe("kubectl accurate", func() {
	It("should configure namespaces", func() {
		kubectlSafe(nil, "create", "ns", "tmpl1")
		kubectlSafe(nil, "label", "ns", "tmpl1", "team=neco")
		kubectlSafe(nil, "create", "ns", "root1")
		_, err := kubectl(nil, "accurate", "template", "set", "root1", "tmpl1")
		Expect(err).To(HaveOccurred())
		kubectlSafe(nil, "accurate", "ns", "set-type", "tmpl1", "template")
		kubectlSafe(nil, "accurate", "ns", "set-type", "tmpl1", "root")
		_, err = kubectl(nil, "accurate", "template", "set", "root1", "tmpl1")
		Expect(err).To(HaveOccurred())
		kubectlSafe(nil, "accurate", "ns", "set-type", "tmpl1", "none")
		kubectlSafe(nil, "accurate", "ns", "set-type", "tmpl1", "template")
		kubectlSafe(nil, "accurate", "template", "set", "root1", "tmpl1")

		Eventually(func() string {
			out, err := kubectl(nil, "get", "ns", "root1", "-o", "json")
			if err != nil {
				return ""
			}
			ns := &corev1.Namespace{}
			if err := json.Unmarshal(out, ns); err != nil {
				return ""
			}
			return ns.Labels["team"]
		}).Should(Equal("neco"))

		_, err = kubectl(nil, "accurate", "ns", "set-type", "tmpl1", "none")
		Expect(err).To(HaveOccurred())
		kubectlSafe(nil, "accurate", "ns", "set-type", "root1", "root")
	})

	It("should propagate resources", func() {
		By("setting up resources")
		kubectlSafe(nil, "create", "ns", "tmpl2")
		kubectlSafe(nil, "annotate", "ns", "tmpl2", "test=foo")
		kubectlSafe(nil, "accurate", "ns", "set-type", "tmpl2", "template")
		kubectlSafe(nil, "create", "ns", "tmpl3")
		kubectlSafe(nil, "accurate", "ns", "set-type", "tmpl3", "template")
		kubectlSafe(nil, "create", "ns", "root2")
		kubectlSafe(nil, "create", "ns", "root3")

		kubectlSafe(roleYAML, "apply", "-f", "-")
		kubectlSafe(nil, "create", "-n", "tmpl3", "secret", "generic", "s1", "--from-literal=foo=bar")
		kubectlSafe(resourceQuota, "apply", "-f", "-")

		By("setting up templates")
		kubectlSafe(nil, "accurate", "template", "set", "tmpl3", "tmpl2")
		kubectlSafe(nil, "accurate", "template", "set", "root2", "tmpl3")

		By("checking propagation from templates")
		kubectlSafe(nil, "annotate", "-n", "tmpl3", "secret", "s1", "accurate.cybozu.com/propagate=update")
		kubectlSafe(nil, "annotate", "-n", "tmpl3", "quota", "rq1", "accurate.cybozu.com/propagate=update")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", "root2", "roles", "role1")
			return err
		}).Should(Succeed())
		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", "root2", "secrets", "s1")
			return err
		}).Should(Succeed())
		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", "root2", "quota", "rq1")
			return err
		}).Should(Succeed())
		Eventually(func() string {
			out, err := kubectl(nil, "get", "ns", "root2", "-o", "json")
			if err != nil {
				return ""
			}
			ns := &corev1.Namespace{}
			if err := json.Unmarshal(out, ns); err != nil {
				return ""
			}
			return ns.Annotations["test"]
		}).Should(Equal("foo"))

		By("unsetting templates")
		kubectlSafe(nil, "accurate", "template", "unset", "root2")

		Eventually(func() error {
			out, err := kubectl(nil, "get", "-n", "root2", "secrets", "-o", "json")
			if err != nil {
				return err
			}
			sl := &corev1.SecretList{}
			if err := json.Unmarshal(out, sl); err != nil {
				return err
			}
			for _, s := range sl.Items {
				if s.Name == "s1" {
					return errors.New("s1 exists")
				}
			}

			out, err = kubectl(nil, "get", "-n", "root2", "quota", "-o", "json")
			if err != nil {
				return err
			}
			rql := &corev1.ResourceQuotaList{}
			if err := json.Unmarshal(out, rql); err != nil {
				return err
			}
			for _, rq := range rql.Items {
				if rq.Name == "rq1" {
					return errors.New("rq1 exists")
				}
			}
			return nil
		}).Should(Succeed())
		kubectlSafe(nil, "get", "-n", "root2", "roles", "role1")

		By("creating sub-namespaces")
		_, err := kubectl(nil, "accurate", "sub", "create", "foo", "root2")
		Expect(err).To(HaveOccurred())
		kubectlSafe(nil, "accurate", "ns", "set-type", "root2", "root")
		kubectlSafe(nil, "accurate", "ns", "set-type", "root3", "root")
		kubectlSafe(nil, "accurate", "sub", "create", "sub1", "root2")
		kubectlSafe(sealedJSON, "apply", "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", "sub1", "secrets", "mysecret")
			return err
		}).Should(Succeed())
		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", "sub1", "roles", "role1")
			return err
		}).Should(Succeed())
	})

	It("should handle sub-namespaces", func() {
		By("preparing root namespaces")
		kubectlSafe(nil, "create", "ns", "subroot1")
		kubectlSafe(nil, "accurate", "ns", "set-type", "subroot1", "root")
		kubectlSafe(nil, "create", "ns", "subroot2")
		kubectlSafe(nil, "accurate", "ns", "set-type", "subroot2", "root")

		By("creating sub-namespaces")
		kubectlSafe(nil, "accurate", "sub", "create", "sn1", "subroot1")
		kubectlSafe(nil, "get", "subnamespaces", "-n", "subroot1", "sn1")
		Eventually(func() error {
			out, err := kubectl(nil, "get", "ns", "sn1", "-o", "json")
			if err != nil {
				return err
			}

			ns := &corev1.Namespace{}
			if err := json.Unmarshal(out, ns); err != nil {
				return err
			}
			if ns.Labels[constants.LabelParent] != "subroot1" {
				return errors.New("wrong parent")
			}
			return nil
		}).Should(Succeed())

		kubectlSafe(nil, "accurate", "sub", "create", "sn2", "sn1")
		kubectlSafe(nil, "get", "subnamespaces", "-n", "sn1", "sn2")
		Eventually(func() error {
			out, err := kubectl(nil, "get", "ns", "sn2", "-o", "json")
			if err != nil {
				return err
			}

			ns := &corev1.Namespace{}
			if err := json.Unmarshal(out, ns); err != nil {
				return err
			}
			if ns.Labels[constants.LabelParent] != "sn1" {
				return errors.New("wrong parent")
			}
			return nil
		}).Should(Succeed())

		By("moving sub-namespaces")
		_, err := kubectl(nil, "accurate", "sub", "move", "sn1", "sn2")
		Expect(err).To(HaveOccurred())

		kubectlSafe(nil, "accurate", "sub", "move", "sn1", "subroot2")
		_, err = kubectl(nil, "get", "subnamespaces", "-n", "subroot1", "sn1")
		Expect(err).To(HaveOccurred())
		kubectlSafe(nil, "get", "subnamespaces", "-n", "subroot2", "sn1")
		out, err := kubectl(nil, "get", "ns", "sn1", "-o", "json")
		Expect(err).NotTo(HaveOccurred())

		sn1 := &corev1.Namespace{}
		err = json.Unmarshal(out, sn1)
		Expect(err).NotTo(HaveOccurred())
		Expect(sn1.Labels[constants.LabelParent]).To(Equal("subroot2"))

		kubectlSafe(nil, "accurate", "sub", "move", "--leave-original", "sn1", "subroot1")
		kubectlSafe(nil, "get", "subnamespaces", "-n", "subroot2", "sn1")
		kubectlSafe(nil, "get", "subnamespaces", "-n", "subroot1", "sn1")

		Eventually(func() string {
			out, err := kubectl(nil, "get", "-n", "subroot2", "subnamespaces", "sn1", "-o", "json")
			if err != nil {
				return ""
			}
			sn := &accuratev1.SubNamespace{}
			if err := json.Unmarshal(out, sn); err != nil {
				return ""
			}
			return string(sn.Status)
		}).Should(Equal("conflict"))

		kubectlSafe(nil, "accurate", "sub", "cut", "sn2")
		_, err = kubectl(nil, "get", "-n", "sn1", "subnamespaces", "sn2")
		Expect(err).To(HaveOccurred())
		out, err = kubectl(nil, "get", "ns", "sn2", "-o", "json")
		Expect(err).NotTo(HaveOccurred())
		sn2 := &corev1.Namespace{}
		err = json.Unmarshal(out, sn2)
		Expect(err).NotTo(HaveOccurred())
		Expect(sn2.Labels).NotTo(HaveKey(constants.LabelParent))

		kubectlSafe(nil, "accurate", "sub", "graft", "sn2", "subroot2")
		out, err = kubectl(nil, "get", "ns", "sn2", "-o", "json")
		Expect(err).NotTo(HaveOccurred())
		sn2 = &corev1.Namespace{}
		err = json.Unmarshal(out, sn2)
		Expect(err).NotTo(HaveOccurred())
		Expect(sn2.Labels).To(HaveKeyWithValue(constants.LabelParent, "subroot2"))
		kubectlSafe(nil, "get", "-n", "subroot2", "subnamespaces", "sn2")

		kubectlSafe(nil, "accurate", "sub", "delete", "sn2")
		Eventually(func() error {
			_, err := kubectl(nil, "get", "ns", "sn2")
			return err
		}).ShouldNot(Succeed())

		_, err = kubectl(nil, "get", "-n", "subroot2", "subnamespaces", "sn2")
		Expect(err).To(HaveOccurred())
	})

	It("should propagate ServiceAccount w/o secrets field", func() {
		kubectlSafe(serviceAccountYAML, "apply", "-f", "-")
		var tokenName string
		Eventually(func() error {
			out, err := kubectl(nil, "-n", "subroot1", "get", "serviceaccounts", "test", "-o", "json")
			if err != nil {
				return err
			}
			sa := &corev1.ServiceAccount{}
			if err := json.Unmarshal(out, sa); err != nil {
				return err
			}
			if len(sa.Secrets) == 0 {
				return errors.New("no token")
			}
			tokenName = sa.Secrets[0].Name
			return nil
		}).Should(Succeed())

		var tokenName2 string
		Eventually(func() error {
			out, err := kubectl(nil, "-n", "sn1", "get", "serviceaccounts", "test", "-o", "json")
			if err != nil {
				return err
			}
			sa := &corev1.ServiceAccount{}
			if err := json.Unmarshal(out, sa); err != nil {
				return err
			}
			if len(sa.Secrets) == 0 {
				return errors.New("no token")
			}
			tokenName2 = sa.Secrets[0].Name
			return nil
		}).Should(Succeed())

		Expect(tokenName2).NotTo(Equal(tokenName))
	})

	It("should run other commands", func() {
		kubectlSafe(nil, "accurate", "list")
		kubectlSafe(nil, "accurate", "sub", "list")
		kubectlSafe(nil, "accurate", "template", "list")
		kubectlSafe(nil, "accurate", "ns", "describe", "tmpl1")
	})
})
