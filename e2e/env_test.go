package e2e

import "os"

var (
	runE2E     = os.Getenv("RUN_E2E") != ""
	kubectlCmd = os.Getenv("KUBECTL")
)
