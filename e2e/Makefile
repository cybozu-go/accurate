KUBERNETES_VERSION = v1.35.0@sha256:4613778f3cfcd10e615029370f5786704559103cf27bef934597ba562b269661 # renovate: kindest/node

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
start: setup $(KUBECTL_ACCURATE)
	kind create cluster --name=accurate --config=$(KIND_CONFIG) --image=kindest/node:$(KUBERNETES_VERSION) --wait 1m
	cd ..; docker build --no-cache -t accurate:dev .
	kind load docker-image accurate:dev --name=accurate
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
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

setup:
	aqua i -l
