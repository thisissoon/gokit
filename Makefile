# General
PKG      := go.soon.build/kit
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)

# Run test suite
.PHONY: test
test:
ifeq ("$(wildcard $(shell which gocov))","")
	go get github.com/axw/gocov/gocov
endif
	gocov test ${PKG_LIST} | gocov report
