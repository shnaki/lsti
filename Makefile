# Go parameters
NAME=lsti
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BIN_DIR=bin

setup:
	go get github.com/golang/lint/golint
	go get golang.org/x/tools/cmd/goimports
	go get github.com/Songmu/make2help/cmd/make2help
deps:
	$(GOGET) github.com/jessevdk/go-flags
	$(GOGET) github.com/jmespath/go-jmespath
	$(GOGET) github.com/mattn/go-zglob

all: test build
build:
	$(GOBUILD) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
run:
	$(GOBUILD)  -v ./...
	./$(NAME)


# Cross compilation
build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -v
build-windows-386:
	GOOS=windows GOARCH=386 $(GOBUILD) -v
build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -v
build-linux-386:
	GOOS=linux GOARCH=386 $(GOBUILD) -v

help:
	@make2help $(MAKEFILE_LIST)
