# Tool versions
CTRL_RUNTIME_VERSION := $(shell awk '/sigs.k8s.io\/controller-runtime/ {print substr($$2, 2)}' go.mod)

# Test tools
BIN_DIR := $(shell pwd)/bin
SUDO = sudo

# Set the shell used to bash for better error handling.
SHELL = /bin/bash
.SHELLFLAGS = -e -o pipefail -c

PATH := $(shell aqua root-dir)/bin:$(PATH)
export PATH

CRD_OPTIONS = "crd:crdVersions=v1,maxDescLen=220"

# for Go
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
SUFFIX =

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

HELM_CRDS_FILE := charts/accurate/templates/generated/crds.yaml
.PHONY: manifests
manifests: setup ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	controller-gen $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="{./api/..., ./controllers/..., ./hooks/...}" output:crd:artifacts:config=config/crd/bases
	echo '{{- include "accurate.crd-check" . }}' > $(HELM_CRDS_FILE)
	echo '{{- if or .Values.crds.enabled .Values.installCRDs }}' >> $(HELM_CRDS_FILE)
	kustomize build config/kustomize-to-helm/overlays/crds | yq e "." -p yaml - >> $(HELM_CRDS_FILE)
	echo '{{- end }}' >> $(HELM_CRDS_FILE)
	kustomize build config/kustomize-to-helm/overlays/templates | yq e "."  -p yaml - > charts/accurate/templates/generated/generated.yaml

.PHONY: generate
generate: setup generate-applyconfigurations generate-conversion ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="{./api/...}"

GO_MODULE = $(shell go list -m)
API_DIRS = $(shell find api -mindepth 2 -type d | sed "s|^|$(shell go list -m)/|" | paste -sd " ")
AC_PKG = internal/applyconfigurations

.PHONY: generate-applyconfigurations
generate-applyconfigurations: setup ## Generate applyconfigurations to support typesafe SSA.
	@echo ">> generating $(AC_PKG)..."
	applyconfiguration-gen \
		--go-header-file 	hack/boilerplate.go.txt \
		--output-dir "$(AC_PKG)" \
		--output-pkg "$(GO_MODULE)/$(AC_PKG)" \
		  $(API_DIRS)

.PHONY: generate-conversion
generate-conversion: setup ## Generate conversion functions to support API conversion.
	@echo ">> generating $(AC_PKG)..."
	conversion-gen \
		--output-file zz_generated.conversion.go \
		$(API_DIRS)

.PHONY: apidoc
apidoc: setup $(wildcard api/*/*_types.go)
	crd-to-markdown --links docs/links.csv -f api/accurate/v1/subnamespace_types.go -n SubNamespace > docs/crd_subnamespace.md

.PHONY: book
book: setup
	rm -rf docs/book
	cd docs; mdbook build

.PHONY: check-generate
check-generate:
	$(MAKE) manifests generate apidoc
	git diff --exit-code --name-only

.PHONY: envtest
envtest: setup
	source <(setup-envtest use -p env); \
		TEST_CONFIG=1 go test -v -count 1 -race ./pkg/config -ginkgo.show-node-events -ginkgo.v -ginkgo.fail-fast
	source <(setup-envtest use -p env); \
		go test -v -count 1 -race ./controllers -ginkgo.show-node-events -ginkgo.v -ginkgo.fail-fast
	source <(setup-envtest use -p env); \
		go test -v -count 1 -race ./hooks/... -ginkgo.show-node-events -ginkgo.v

.PHONY: lint
lint: setup
	golangci-lint run ./... -v

.PHONY: lint-fix
lint-fix: setup
	golangci-lint run ./... -v --fix

.PHONY: test
test: 
	go test -v -count 1 -race ./api/... ./internal/... ./pkg/...
	go install ./...
	go vet ./...
	test -z $$(gofmt -s -l . | tee /dev/stderr)

##@ Build

.PHONY: build
build:
	mkdir -p bin
	GOBIN=$(shell pwd)/bin go install ./cmd/...

.PHONY: release-build
release-build: setup
	goreleaser build --snapshot --clean

##@ Tools

setup:
	aqua policy allow ./aqua-policy.yaml
	aqua i -l
