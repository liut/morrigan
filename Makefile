.SILENT :
.PHONY: docs

NAME:=morrigan
ROOF:=github.com/liut/morrigan
DATE := $(shell date '+%Y%m%d')
TAG:=$(shell git describe --tags --always)
GO=$(shell which go)
GOMOD=$(shell echo "$${GO111MODULE:-auto}")
WEBAPIS=$(shell find pkg/web -type f \( -name "*.go" ! -name "*_test.go" \) -print )
LDFLAGS:=-X $(ROOF)/pkg/settings.name=$(NAME) -X $(ROOF)/pkg/settings.version=$(DATE)-$(TAG)

MDs=$(shell find docs -type f \( -name "*.yaml" ! -name "swagger.yaml" \) -print )
SPEC=7

help:
	echo "make modcodegen"

docs:
	$(info docs: $(MDs))

codegen:
	mkdir -p ./pkg/models ./pkg/services/stores ./pkg/web
	for name in $(MDs); do \
		echo $${name}; \
		GO111MODULE=on $(GO) run -tags=codegen ./scripts/codegen -spec $(SPEC) $${name} ; \
	done

generate:
	GO111MODULE=$(GOMOD) $(GO) generate ./...

modcodegen: # deprecated
	mkdir -p ./pkg/models
	for name in $(MDs); do \
		echo $${name}; \
		GO111MODULE=$(GOMOD) $(GO) run -tags=sdkcodegen ./scripts/sdkcodegen docs/$${name} ./pkg/models/$${name}.go ; \
	done

vet:
	echo "Checking ./pkg/... ./cmd/... , with GOMOD=$(GOMOD)"
	GO111MODULE=$(GOMOD) $(GO) vet -all ./pkg/...

deps:
	GO111MODULE=on $(GO) install golang.org/x/tools/cmd/goimports@latest
	GO111MODULE=on $(GO) install github.com/ddollar/forego@latest
	GO111MODULE=on $(GO) install github.com/liut/rerun@latest
	GO111MODULE=on $(GO) install github.com/swaggo/swag/cmd/swag@latest
	GO111MODULE=on $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	GO111MODULE=$(GOMOD) golangci-lint --disable structcheck run ./...

clean:
	echo "Cleaning dist"
	rm -rf dist
	rm -f ./$(NAME)-*

showver:
	echo "version: $(TAG)"

dist/linux_amd64/$(NAME): $(SOURCES) showver
	mkdir -p dist/linux_amd64 && GO111MODULE=$(GOMOD) GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) -s -w" -o dist/linux_amd64/$(NAME) .

dist/darwin_amd64/$(NAME): $(SOURCES) showver
	echo "Building $(NAME) of darwin"
	mkdir -p dist/darwin_amd64 && GO111MODULE=$(GOMOD) GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) -w" -o dist/darwin_amd64/$(NAME) .

dist: vet dist/linux_amd64/$(NAME) dist/darwin_amd64/$(NAME)

package: dist
	echo "Packaging $(NAME)"
	ls dist/linux_amd64 | xargs tar -cvJf $(NAME)-linux-amd64-$(TAG).tar.xz -C dist/linux_amd64
	ls dist/darwin_amd64 | xargs tar -cvJf $(NAME)-darwin-amd64-$(TAG).tar.xz -C dist/darwin_amd64

docs/swagger.json: $(WEBAPIS)
	GO111MODULE=on swag init -g ./pkg/web/docs.go -d ./ --ot json,yaml --parseDependency --parseInternal

touch-web-api:
	touch pkg/web/docs.go

gen-apidoc: touch-web-api docs/swagger.json
