VERSION ?= latest
GOVERSION ?= 1.14
BUILD_DIR := bin
CMD := saymyname
USER ?= leodido
NAME ?= saymyname-prow-plugin
PKG ?= github.com/$(USER)/$(NAME)


all: vendor build

.PHONY: vendor
vendor:
	@GO111MODULE=on go mod download
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor

.PHONY: clean
clean:
	@GOOS=linux go clean -i -x ./...
	rm -f $(PWD)/$(BUILD_DIR)/$(CMD)

.PHONY: mkdir
mkdir:
	@mkdir -p $(BUILD_DIR)

.PHONY: compile
compile: mkdir
	@docker run \
		-v $(PWD):/go/src/$(PKG) \
		-w /go/src/$(PKG) \
		-e GOOS=linux -e GOARCH=amd64 -e CGO_ENABLED=0 -e GOFLAGS=-mod=vendor \
		golang:$(GOVERSION)-alpine3.12 go build -o $(BUILD_DIR)/$(CMD) .

.PHONY: build
build: compile
	@docker build -t $(USER)/$(NAME):$(VERSION) .

.PHONY: push
push:
	@docker push $(USER)/$(NAME):$(VERSION)