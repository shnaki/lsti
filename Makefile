# Go parameters
NAME=lsti
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BIN_DIR=bin

## Set up dev tools
setup:
	go get github.com/golang/lint/golint
	go get golang.org/x/tools/cmd/goimports
	go get github.com/Songmu/make2help/cmd/make2help
.PHONY: setup

## Install dependencies
deps:
	$(GOGET) github.com/jessevdk/go-flags
	$(GOGET) github.com/jmespath/go-jmespath
	$(GOGET) github.com/mattn/go-zglob
	$(GOGET) github.com/olekukonko/tablewriter
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

## Show help
help:
	@make2help $(MAKEFILE_LIST)
.PHONY: help
