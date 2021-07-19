KIND_VERSION = 0.11.1
KUBERNETES_VERSION = 1.20.7
CERT_MANAGER_VERSION = 1.4.0
SEALED_SECRET_VERSION = 0.16.0

KIND := $(dir $(shell pwd))/bin/kind
KUBECTL := $(dir $(shell pwd))/bin/kubectl
KUSTOMIZE := $(dir $(shell pwd))/bin/kustomize
KUBESEAL := $(dir $(shell pwd))/bin/kubeseal
KUBECTL_INNU := $(dir $(shell pwd))/bin/kubectl-innu
KUBECONFIG := $(shell pwd)/.kubeconfig
KIND_CONFIG = kind-config.yaml
export KUBECTL KUBECONFIG


.PHONY: help
help:
	@echo "Choose one of the following target"
	@echo
	@echo "start      Start kind cluster and install Innu"
	@echo "test       Run e2e tests"
	@echo "logs       Save logs as logs.tar.gz"
	@echo "stop       Stop the kind cluster"

.PHONY: start
start: $(KIND) $(KUBECTL) $(KUSTOMIZE) $(KUBESEAL) $(KUBECTL_INNU)
	$(KIND) create cluster --name=innu --config=$(KIND_CONFIG) --image=kindest/node:v$(KUBERNETES_VERSION) --wait 1m
	cd ..; docker build --no-cache -t innu:dev .
	$(KIND) load docker-image innu:dev --name=innu
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) -n cert-manager wait --for=condition=available --timeout=180s --all deployments
	$(KUSTOMIZE) build . | $(KUBECTL) apply -f -
	$(KUBECTL) -n innu wait --for=condition=available --timeout=180s --all deployments
	$(KUBECTL) apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v$(SEALED_SECRET_VERSION)/controller.yaml
	$(KUBECTL) -n kube-system wait --for=condition=available --timeout=180s --all deployments
	$(KUBECTL) apply -f clusterrole_for_sealedsecrets.yaml
	rm -f sealed.json
	$(KUBECTL) -n root2 create secret generic mysecret --dry-run=client --from-literal=foo=bar -o json | \
		$(KUBESEAL) | jq '. * {metadata: {annotations: {"innu.cybozu.com/propagate-generated": "create"}}}' > sealed.json

.PHONY: test
test:
	env PATH=$$(pwd)/../bin:$$PATH RUN_E2E=1 \
		go test -v -race . -ginkgo.progress -ginkgo.v -ginkgo.failFast

.PHONY: test-upgrade
test-upgrade:
	env PATH=$$(pwd)/../bin:$$PATH RUN_E2E=1 UPGRADE=1 \
		go test -v -race . -ginkgo.progress -ginkgo.v -ginkgo.failFast

.PHONY: logs
logs:
	rm -rf logs.tar.gz logs
	$(KIND) export logs --name=innu ./logs
	tar czf logs.tar.gz logs
	rm -rf logs

.PHONY: stop
stop: $(KIND) 
	$(KIND) delete cluster --name=innu
	-docker image rm innu:dev
	-docker image prune -f

$(KIND):
	mkdir -p ../bin
	curl -sfL -o $@ https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-linux-amd64
	chmod a+x $@

$(KUBECTL):
	mkdir -p ../bin
	curl -sfL -o $@ https://dl.k8s.io/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $@

$(KUSTOMIZE):
	$(MAKE) -C .. kustomize

$(KUBESEAL):
	mkdir -p ../bin
	curl -sfL -o $@ https://github.com/bitnami-labs/sealed-secrets/releases/download/v$(SEALED_SECRET_VERSION)/kubeseal-linux-amd64
	chmod a+x $@

$(KUBECTL_INNU): $(wildcard ../cmd/kubectl-innu/*/*.go)
	mkdir -p ../bin
	cd ..; GOBIN=$$(pwd)/bin go install ./cmd/kubectl-innu
