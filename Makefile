# General
PKG      := go.soon.build/kit
MODULES  := $(shell ls -p | grep /)
CWD      := $(shell pwd)

# Download dependencies for all modules
.PHONY: download
download:
	@$(foreach module,$(MODULES),cd $(CWD)/$(module) && go mod download;)

# Run test suite
.PHONY: test
test:
ifeq ("$(wildcard $(shell which gocov))","")
	go install github.com/axw/gocov/gocov@v1.1.0
endif
	@$(foreach module,$(MODULES),cd $(CWD)/$(module) && gocov test ./... | gocov report;)

# Run integration tests with gcloud pubsub
testgcloud:
	@$(foreach module,$(MODULES),cd $(CWD)/$(module) && gocov test --tags gcloud ./... | gocov report;)

lint:
	@$(foreach module,$(MODULES),cd $(CWD)/$(module) && golangci-lint run;)
