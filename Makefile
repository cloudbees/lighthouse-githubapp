#
# Copyright (C) CloudBees
#
SHELL := /bin/bash
NAME := lighthouse-githubapp
PACKAGE_NAME := github.com/cloudbees/lighthouse-githubapp
GO := GO111MODULE=on GO15VENDOREXPERIMENT=1 go
GO_NOMOD := GO111MODULE=off go

# set dev version unless VERSION is explicitly set via environment
VERSION ?= $(shell echo "$$(git describe --abbrev=0 --tags 2>/dev/null)-dev+$(REV)" | sed 's/^v//')

#ROOT_PACKAGE := $(shell $(GO) list .)
ROOT_PACKAGE := github.com/cloudbees/lighthouse-githubapp
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
#PACKAGE_DIRS := pkg cmd
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/ | grep -v e2e)
PEGOMOCK_PACKAGE := github.com/petergtz/pegomock
#GO_DEPENDENCIES := $(call rwildcard,pkg/,*.go) main.go
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')

REV        := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
SHA1       := $(shell git rev-parse HEAD 2> /dev/null || echo 'unknow')
BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
BUILDFLAGS := -ldflags \
  " -X $(ROOT_PACKAGE)/pkg/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/pkg/version.Revision=$(REV)\
		-X $(ROOT_PACKAGE)/pkg/version.Sha1=$(SHA1)\
		-X $(ROOT_PACKAGE)/pkg/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/pkg/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/pkg/version.GoVersion='$(GO_VERSION)'"
CGO_ENABLED = 0
BUILDTAGS :=

CLIENTSET_GENERATOR_VERSION := kubernetes-1.11.3
CODE_GEN_BIN_NAME := codegen
CODE_GEN_GO_DEPENDENCIES := $(call rwildcard,cmd/codegen/,*.go)
CODE_GEN_BUILDFLAGS :=
ifdef DEBUG
CODE_GEN_BUILDFLAGS := -gcflags "all=-N -l" $(CODE_GEN_BUILDFLAGS)
endif

# Expose pprof profiling service at localhost:6060
#BUILDTAGS := --tags pprof

export GITHUB_ACCESS_TOKEN=$(shell cat /builder/home/github.token 2> /dev/null || echo 'unset')

SOURCE_DIR ?= .
DESIGN_DIR=pkg/design
DESIGNS := ./pkg/design/*.go

GOAGEN_BIN=goagen
GO_BINDATA_BIN=go-bindata

GOPATH1=$(firstword $(subst :, ,$(GOPATH)))

export PATH := $(PATH):$(GOPATH1)/bin

build: $(GO_DEPENDENCIES)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$(NAME)

all: version check

check: fmt build test

version:
	echo "Go version: $(GO_VERSION)"

test:
	DISABLE_SSO=true CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(PACKAGE_DIRS)

testv:
	DISABLE_SSO=true CGO_ENABLED=$(CGO_ENABLED) $(GO) test -test.v $(PACKAGE_DIRS)

testrich:
	DISABLE_SSO=true CGO_ENABLED=$(CGO_ENABLED) richgo test -test.v $(PACKAGE_DIRS)

test1:
	DISABLE_SSO=true CGO_ENABLED=$(CGO_ENABLED) $(GO) test  -count=1  -short ./... -test.v  -run $(TEST)


generate-e2e-test-token:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run scripts/setup-e2e-test-user.go

.PHONY:e2e
e2e:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test github.com/cloudbees/lighthouse-githubapp/e2e -v -timeout 15m

install: $(GO_DEPENDENCIES)
	GOBIN=${GOPATH1}/bin $(GO) install $(BUILDFLAGS) main.go

fmt:
	@FORMATTED=`$(GO) fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

arm:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$(NAME)-arm main.go

win:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$(NAME).exe main.go

generate: generate-goa generate-mocks

generate-goa: generate-goagen templates

.PHONY: generate-goagen generate-mocks
## Generate GOA sources. Only necessary after clean of if changed `design` folder.
generate-goagen: $(GOAGEN_BIN) $(GO_BINDATA_BIN) $(DESIGNS)
	$(GOAGEN_BIN) app -d ${PACKAGE_NAME}/${DESIGN_DIR} -o pkg/controller
	$(GOAGEN_BIN) controller -d ${PACKAGE_NAME}/${DESIGN_DIR} -o pkg/controller/ --pkg controller --app-pkg app
	$(GOAGEN_BIN) swagger -d ${PACKAGE_NAME}/${DESIGN_DIR} -o pkg
	$(GOAGEN_BIN) schema -d ${PACKAGE_NAME}/${DESIGN_DIR} -o pkg

generate-mocks:
	$(GO) get $(PEGOMOCK_PACKAGE)/...
	$(GO) generate -run="pegomock" ./...


.PHONY: clean-generated
## Removes all generated code.
clean-generated:
	-rm -rf ./pkg/app
	-rm ./pkg/swagger/*.json ./pkg/swagger/*.yaml

$(GOAGEN_BIN):
	rm -f ~/go/bin/goagen
	echo "getting goagen"
	$(GO) get github.com/goadesign/goa/goagen@v1.4.1
	which goagen

$(GO_BINDATA_BIN): $(VENDOR_DIR)
	$(GO_NOMOD) get -u github.com/jteeuwen/go-bindata/go-bindata


templates: schema/bindata.go swagger/bindata.go

swagger/bindata.go: $(GO_BINDATA_BIN) pkg/swagger/swagger.json pkg/swagger/swagger.yaml
	$(GO_BINDATA_BIN) \
		-o pkg/swagger/bindata.go \
		-pkg swagger \
		-prefix '' \
		-nocompress \
		pkg/swagger

schema/bindata.go: $(GO_BINDATA_BIN) pkg/schema/schema.json
	$(GO_BINDATA_BIN) \
		-o pkg/schema/bindata.go \
		-pkg schema \
		-prefix '' \
		-nocompress \
		pkg/schema

release: check
	rm -rf build release && mkdir build release
	for os in linux darwin ; do \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$$os/$(NAME) main.go ; \
	done
	# jxt runs in kubernetes so shouldn't need windows binaries
	# CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-windows-amd64.exe main.go
	# zip --junk-paths release/$(NAME)-windows-amd64.zip build/$(NAME)-windows-amd64.exe README.md LICENSE
	# CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/arm/$(NAME) main.go
	chmod +x build/darwin/$(NAME)
	chmod +x build/linux/$(NAME)
	# chmod +x build/arm/$(NAME)

	cp ./build/linux/jxt ./build/jxt

	cd ./build/darwin; tar -zcvf ../../release/jxt-darwin-amd64.tar.gz jxt
	cd ./build/linux; tar -zcvf ../../release/jxt-linux-amd64.tar.gz jxt
	# cd ./build/arm; tar -zcvf ../../release/jxt-linux-arm.tar.gz jxt

	go get -u github.com/progrium/gh-release
	gh-release checksums sha256
	gh-release create cloudbees/lighthouse-githubapp $(VERSION) master $(VERSION)

clean:
	rm -rf build release

linux:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$(NAME)-linux-amd64 main.go

linux-dbg:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build -gcflags "-N -l" $(BUILDFLAGS) -o build/$(NAME)-linux-amd64-dbg main.go

docker: linux
	docker build -t cloudbees/lighthouse-githubapp .

richgo:
	go get -u github.com/kyoh86/richgo


# code generation
build-codegen: build/$(CODE_GEN_BIN_NAME) ## Build the code generator

build/$(CODE_GEN_BIN_NAME): $(CODE_GEN_GO_DEPENDENCIES)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(CODE_GEN_BUILDFLAGS) -o build/$(CODE_GEN_BIN_NAME) cmd/codegen/codegen.go

test-codegen: ## Test the code geneator
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v -short ./cmd/codegen/...

generate-client: codegen-clientset fmt ## Generate the client

modtidy:
	go mod tidy

mod: modtidy build

codegen-clientset: build-codegen ## Generate the k8s types and clients
	@echo "Generating Kubernetes Clients for pkg/aps/v1alpha1 in pkg/apsclient for ssa.googlesource.com:v1alpha1"
	./build/$(CODE_GEN_BIN_NAME) --generator-version $(CLIENTSET_GENERATOR_VERSION) clientset --output-package=pkg/aps --input-package=pkg/aps --group-with-version=ssa.googlesource.com:v1alpha1

.PHONY: release clean arm


