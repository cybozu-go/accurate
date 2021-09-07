# Tool versions
CTRL_TOOLS_VERSION=0.6.1
CTRL_RUNTIME_VERSION := $(shell awk '/sigs.k8s.io\/controller-runtime/ {print substr($$2, 2)}' go.mod)
HELM_VERSION = 3.6.3
HELM_DOCS_VERSION = 1.5.0
CRD_TO_MARKDOWN_VERSION = 0.0.3
MDBOOK_VERSION = 0.4.10

# Test tools
BIN_DIR := $(shell pwd)/bin
STATICCHECK := $(BIN_DIR)/staticcheck
NILERR := $(BIN_DIR)/nilerr
SUDO = sudo

# Set the shell used to bash for better error handling.
SHELL = /bin/bash
.SHELLFLAGS = -e -o pipefail -c

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

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: apidoc
apidoc: crd-to-markdown $(wildcard api/*/*_types.go)
	$(CRD_TO_MARKDOWN) --links docs/links.csv -f api/v1/subnamespace_types.go -n SubNamespace > docs/crd_subnamespace.md

.PHONY: book
book: mdbook
	rm -rf docs/book
	cd docs; $(MDBOOK) build

.PHONY: check-generate
check-generate:
	$(MAKE) manifests generate apidoc
	git diff --exit-code --name-only

.PHONY: envtest
envtest: setup-envtest
	source <($(SETUP_ENVTEST) use -p env); \
		TEST_CONFIG=1 go test -v -count 1 -race ./pkg/config -ginkgo.progress -ginkgo.v -ginkgo.failFast
	source <($(SETUP_ENVTEST) use -p env); \
		go test -v -count 1 -race ./controllers -ginkgo.progress -ginkgo.v -ginkgo.failFast
	source <($(SETUP_ENVTEST) use -p env); \
		go test -v -count 1 -race ./hooks -ginkgo.progress -ginkgo.v

.PHONY: test
test: test-tools
	go test -v -count 1 -race ./pkg/...
	go install ./...
	go vet ./...
	test -z $$(gofmt -s -l . | tee /dev/stderr)
	$(STATICCHECK) ./...
	$(NILERR) ./...

##@ Build

.PHONY: build
build:
	mkdir -p bin
	GOBIN=$(shell pwd)/bin go install ./cmd/...

.PHONY: release-build
release-build:
	rm -rf build
	mkdir -p build
	$(MAKE) kubectl-accurate GOOS=windows GOARCH=amd64 SUFFIX=.exe
	$(MAKE) kubectl-accurate GOOS=darwin GOARCH=amd64
	$(MAKE) kubectl-accurate GOOS=darwin GOARCH=arm64
	$(MAKE) kubectl-accurate GOOS=linux GOARCH=amd64
	$(MAKE) kubectl-accurate GOOS=linux GOARCH=arm64

.PHONY: kubectl-accurate
kubectl-accurate: build/kubectl-accurate-$(GOOS)-$(GOARCH)$(SUFFIX)

build/kubectl-accurate-$(GOOS)-$(GOARCH)$(SUFFIX):
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $@ ./cmd/kubectl-accurate

##@ Tools

CONTROLLER_GEN := $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CTRL_TOOLS_VERSION))

SETUP_ENVTEST := $(shell pwd)/bin/setup-envtest
.PHONY: setup-envtest
setup-envtest: $(SETUP_ENVTEST) ## Download setup-envtest locally if necessary
$(SETUP_ENVTEST):
	# see https://github.com/kubernetes-sigs/controller-runtime/tree/master/tools/setup-envtest
	GOBIN=$(shell pwd)/bin go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

HELM := $(shell pwd)/bin/helm
.PHONY: helm
helm: $(HELM) ## Download helm locally if necessary.

$(HELM):
	mkdir -p $(BIN_DIR)
	curl -L -sS https://get.helm.sh/helm-v$(HELM_VERSION)-linux-amd64.tar.gz \
	  | tar xz -C $(BIN_DIR) --strip-components 1 linux-amd64/helm

HELM_DOCS := $(shell pwd)/bin/helm-docs
.PHONY: helm-docs
helm-docs: $(HELM_DOCS) ## Download helm-docs locally if necessary.

$(HELM_DOCS):
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/norwoodj/helm-docs/cmd/helm-docs@v$(HELM_DOCS_VERSION)

CRD_TO_MARKDOWN := $(shell pwd)/bin/crd-to-markdown
.PHONY: crd-to-markdown
crd-to-markdown: ## Download crd-to-markdown locally if necessary.
	$(call go-get-tool,$(CRD_TO_MARKDOWN),github.com/clamoriniere/crd-to-markdown@v$(CRD_TO_MARKDOWN_VERSION))

MDBOOK := $(shell pwd)/bin/mdbook
.PHONY: mdbook
mdbook: $(MDBOOK) ## Donwload mdbook locally if necessary
$(MDBOOK):
	mkdir -p bin
	curl -fsL https://github.com/rust-lang/mdBook/releases/download/v$(MDBOOK_VERSION)/mdbook-v$(MDBOOK_VERSION)-x86_64-unknown-linux-gnu.tar.gz | tar -C bin -xzf -

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef

.PHONY: test-tools
test-tools: $(STATICCHECK) $(NILERR)

$(STATICCHECK):
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install honnef.co/go/tools/cmd/staticcheck@latest

$(NILERR):
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/gostaticanalysis/nilerr/cmd/nilerr@latest
