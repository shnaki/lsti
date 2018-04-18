# Go parameters
NAME=lsti
VERSION=1.0.1
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
DIST_DIR=dist

## Show help
help: setup-help
	@make2help $(MAKEFILE_LIST)
.PHONY: help

## Set up dev tools
setup: setup-help setup-goxz
.PHONY: setup

## Set up goxz
setup-goxz:
	$(GOGET) github.com/Songmu/goxz/cmd/goxz
.PHONY: setup-goxz

## Set up make2help
setup-help:
	$(GOGET) github.com/Songmu/make2help/cmd/make2help
.PHONY: setup-help

## Install dependencies
deps:
	$(GOGET) github.com/jessevdk/go-flags
	$(GOGET) github.com/jmespath/go-jmespath
	$(GOGET) github.com/mattn/go-zglob
	$(GOGET) github.com/olekukonko/tablewriter
	$(GOGET) github.com/russross/blackfriday
.PHONY: deps

## Run tests and build binary
all: test build
.PHONY: all

## Build binary
build:
	$(GOBUILD) -v
.PHONY: build

## Run tests
test:
	$(GOTEST) -v ./...
.PHONY: test

## Remove binaries
clean:
	$(GOCLEAN)
	rm -rf $(DIST_DIR)
.PHONY: clean

## Builds the binary and executes the application consequently
run:
	$(GOBUILD)  -v ./...
	./$(NAME)
.PHONY: run


# Cross compilation
## Build binary for windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -v

## Build binary for windows-386
build-windows-386:
	GOOS=windows GOARCH=386 $(GOBUILD) -v

## Build binary for linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -v

## Build binary for linux-386
build-linux-386:
	GOOS=linux GOARCH=386 $(GOBUILD) -v

## Release binaries
release: deps setup-goxz
	goxz -pv=$(VERSION) -os=windows,darwin,linux -arch=amd64,386 -d=$(DIST_DIR)
.PHONY: release
