SHELL := /usr/bin/env bash
GO ?= $$(which go)
REPO_ROOT := $$(git rev-parse --show-toplevel)
CHAPTERS := $(wildcard chapter*)
BIN_DIR := $(REPO_ROOT)/bin
SCRIPTS_DIR := $(REPO_ROOT)/scripts
LAUNCHER := $(SCRIPTS_DIR)/launcher.sh

$(CHAPTERS): 
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 $(GO) build -o "$(BIN_DIR)/$@" "$(REPO_ROOT)/$@" && \
		RUNFILES_DIR=$(BIN_DIR) $(LAUNCHER) "./bin/$@"

.PHONY: $(CHAPTERS)

