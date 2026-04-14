package e2e

import (
	"bytes"
	"fmt"
	"os/exec"

	. "github.com/onsi/gomega"
)

func kubectl(input []byte, args ...string) ([]byte, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := exec.Command("kubectl", args...) // #nosec G204 -- args are static/test-controlled
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), nil
	}
	return nil, fmt.Errorf("kubectl failed with %w: stderr=%s", err, stderr.String())
}

func kubectlSafe(input []byte, args ...string) []byte {
	out, err := kubectl(input, args...)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return out
}
