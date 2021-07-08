package controllers

import (
	"strings"

	"github.com/cybozu-go/innu/pkg/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func cloneResource(res *unstructured.Unstructured, ns string) *unstructured.Unstructured {
	c := res.DeepCopy()
	delete(c.Object, "metadata")
	delete(c.Object, "status")
	c.SetNamespace(ns)
	c.SetName(res.GetName())
	labels := make(map[string]string)
	for k, v := range res.GetLabels() {
		if strings.Contains(k, "kubernetes.io/") {
			continue
		}
		labels[k] = v
	}
	labels[constants.LabelCreatedBy] = constants.CreatedBy
	c.SetLabels(labels)
	annotations := make(map[string]string)
	for k, v := range res.GetAnnotations() {
		if strings.Contains(k, "kubernetes.io/") {
			continue
		}
		annotations[k] = v
	}
	annotations[constants.AnnFrom] = res.GetNamespace()
	c.SetAnnotations(annotations)

	return c
}
