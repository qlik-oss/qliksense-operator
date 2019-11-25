SHELL = bash


COMMIT ?= $(shell git rev-parse --short HEAD)
VERSION ?= $(shell git describe --tags 2> /dev/null || echo v0)
PERMALINK ?= $(shell git name-rev --name-only --tags --no-undefined HEAD &> /dev/null && echo latest || echo canary)
REPOSITORY = qlik/qliksense-operator

BINDIR = bin/qliksense-operator

PKG = github.com/qlik-oss/qliksense-operator
MIXIN = qliksense-operator


LDFLAGS = -w -X $(PKG)/pkg.Version=$(VERSION) -X $(PKG)/pkg.Commit=$(COMMIT)
XBUILD = CGO_ENABLED=0 go build -a -tags netgo -ldflags '$(LDFLAGS)'

CLIENT_PLATFORM ?= $(shell go env GOOS)
CLIENT_ARCH ?= $(shell go env GOARCH)
RUNTIME_PLATFORM ?= linux
RUNTIME_ARCH ?= amd64
SUPPORTED_PLATFORMS = linux darwin windows
SUPPORTED_ARCHES = amd64


ifeq ($(CLIENT_PLATFORM),windows)
FILE_EXT=.exe
else ifeq ($(RUNTIME_PLATFORM),windows)
FILE_EXT=.exe
else
FILE_EXT=
endif



REGISTRY ?= $(USER)

build: build-client

build-client: generate
	mkdir -p $(BINDIR)
	go build -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(MIXIN)$(FILE_EXT)

generate: packr2
	go generate ./...

HAS_PACKR2 := $(shell command -v packr2)
packr2:
ifndef HAS_PACKR2
	go get -u github.com/gobuffalo/packr/v2/packr2
endif

xbuild-all: generate
	$(foreach OS, $(SUPPORTED_PLATFORMS), \
		$(foreach ARCH, $(SUPPORTED_ARCHES), \
				$(MAKE) $(MAKE_OPTS) CLIENT_PLATFORM=$(OS) CLIENT_ARCH=$(ARCH) MIXIN=$(MIXIN) xbuild; \
		))

xbuild: $(BINDIR)/$(VERSION)/$(MIXIN)-$(CLIENT_PLATFORM)-$(CLIENT_ARCH)$(FILE_EXT)
$(BINDIR)/$(VERSION)/$(MIXIN)-$(CLIENT_PLATFORM)-$(CLIENT_ARCH)$(FILE_EXT):
	mkdir -p $(dir $@)
	GOOS=$(CLIENT_PLATFORM) GOARCH=$(CLIENT_ARCH) $(XBUILD) -o $@ 

vendor: dep
	dep ensure

test: test-unit
	$(BINDIR)/$(MIXIN)$(FILE_EXT) version

test-unit: build
	go test ./...

clean: clean-packr
	-rm -fr bin/

build-docker:
	docker build $(BINDIR) -t qlik/qliksense-operator:$(VERSION)

docker-push: build-docker
	docker push $(REPOSITORY)
