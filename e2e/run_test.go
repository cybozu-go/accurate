package e2e

import (
	"bytes"
	"fmt"
	"os/exec"

	. "github.com/onsi/gomega"
	dto "github.com/prometheus/client_model/go"
)

func kubectl(input []byte, args ...string) ([]byte, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := exec.Command(kubectlCmd, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), nil
	}
	return nil, fmt.Errorf("kubectl failed with %s: stderr=%s", err, stderr)
}

func kubectlSafe(input []byte, args ...string) []byte {
	out, err := kubectl(input, args...)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return out
}

func runInPod(args ...string) ([]byte, error) {
	a := append([]string{"exec", "client", "--"}, args...)
	return kubectl(nil, a...)
}

func findMetric(mf *dto.MetricFamily, labels map[string]string) *dto.Metric {
OUTER:
	for _, m := range mf.Metric {
		having := make(map[string]string)
		for _, p := range m.Label {
			having[*p.Name] = *p.Value
		}
		for k, v := range labels {
			if having[k] != v {
				continue OUTER
			}
		}
		return m
	}
	return nil
}
