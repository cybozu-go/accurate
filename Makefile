

BIN_DIR := $(shell pwd)/bin
STATICCHECK = $(BIN_DIR)/staticcheck
NILERR = $(BIN_DIR)/nilerr

.PHONY: all
all: test

.PHONY: test
test: test-tools
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	$(STATICCHECK) ./...
	test -z "$$($(NILERR) ./... 2>&1 | tee /dev/stderr)"
	go install ./...
	go test -race -v ./...
	go vet ./...

.PHONY: test-tools
test-tools: $(STATICCHECK) $(NILERR)

$(STATICCHECK):
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install honnef.co/go/tools/cmd/staticcheck@latest

$(NILERR):
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/gostaticanalysis/nilerr/cmd/nilerr@latest
