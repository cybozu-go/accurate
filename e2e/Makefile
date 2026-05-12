KUBERNETES_VERSION = v1.35.0@sha256:4613778f3cfcd10e615029370f5786704559103cf27bef934597ba562b269661 # renovate: kindest/node
# renovate: datasource=helm registryUrl=https://charts.jetstack.io depName=cert-manager
CERT_MANAGER_VERSION = v1.20.2
CERT_MANAGER_GPG_KEY_URL = https://cert-manager.io/public-keys/cert-manager-keyring-2021-09-20-1020CF3C033D4F35BAE1C19E1226061C665DF13E.gpg

PROJECT_DIR := $(CURDIR)/../
CHART_DIR := $(PROJECT_DIR)/tmp-charts

KUBECTL_ACCURATE := $(dir $(shell pwd))/bin/kubectl-accurate
KUBECONFIG := $(shell pwd)/.kubeconfig
KIND_CONFIG = kind-config.yaml
export KUBECTL KUBECONFIG

PATH := $(shell aqua root-dir)/bin:$(PATH)"
export PATH

.PHONY: help
help:
	@echo "Choose one of the following target"
	@echo
	@echo "start      Start kind cluster and install accurate"
	@echo "test       Run e2e tests"
	@echo "logs       Save logs as logs.tar.gz"
	@echo "stop       Stop the kind cluster"

.PHONY: start
start: setup $(KUBECTL_ACCURATE) $(CHART_DIR)/cert-manager/Chart.yaml
	kind create cluster --name=accurate --config=$(KIND_CONFIG) --image=kindest/node:$(KUBERNETES_VERSION) --wait 1m
	cd ..; docker build --no-cache -t accurate:dev .
	kind load docker-image accurate:dev --name=accurate
	helm install cert-manager $(CHART_DIR)/cert-manager \
		--namespace cert-manager --create-namespace --set crds.enabled=true
	kubectl -n cert-manager wait --for=condition=available --timeout=180s --all deployments
	helm install --create-namespace --namespace accurate accurate ../charts/accurate -f values.yaml
	kubectl -n accurate wait --for=condition=available --timeout=180s --all deployments
	helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
	helm repo update
	helm install sealed-secrets-controller --namespace kube-system sealed-secrets/sealed-secrets
	kubectl -n kube-system wait --for=condition=available --timeout=180s --all deployments
	# wait for a secret to be created
	sleep 5
	kubectl -n root2 create secret generic mysecret --dry-run=client --from-literal=foo=bar -o json | \
		kubeseal | jq '. * {metadata: {annotations: {"accurate.cybozu.com/propagate-generated": "create"}}}' > sealed.json

.PHONY: test
test:
	env PATH=$$(pwd)/../bin:$$PATH RUN_E2E=1 \
		go test -v -race . -ginkgo.show-node-events -ginkgo.v -ginkgo.fail-fast

.PHONY: logs
logs:
	rm -rf logs.tar.gz logs
	kind export logs --name=accurate ./logs
	tar czf logs.tar.gz logs
	rm -rf logs

.PHONY: stop
stop: setup
	kind delete cluster --name=accurate
	-docker image rm accurate:dev
	-docker image prune -f

$(KUBECTL_ACCURATE): $(wildcard ../cmd/kubectl-accurate/*/*.go)
	mkdir -p ../bin
	cd ..; GOBIN=$$(pwd)/bin go install ./cmd/kubectl-accurate

$(CHART_DIR):
	mkdir -p $(CHART_DIR)

$(CHART_DIR)/cert-manager-keyring.gpg: | $(CHART_DIR)
	curl -fsSL -o $(CHART_DIR)/cert-manager-keyring.gpg $(CERT_MANAGER_GPG_KEY_URL)

$(CHART_DIR)/cert-manager/Chart.yaml: $(CHART_DIR)/cert-manager-keyring.gpg
	helm pull oci://quay.io/jetstack/charts/cert-manager \
		--version $(CERT_MANAGER_VERSION) \
		--verify --keyring $(CHART_DIR)/cert-manager-keyring.gpg \
		--untar --untardir $(CHART_DIR)

setup:
	aqua i -l
