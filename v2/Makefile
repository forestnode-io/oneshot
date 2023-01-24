APP_NAME ?= oneshot
IMAGE_REGISTRY ?= docker.io/raphaelreyna
GIT_REPO ?= github.com/raphaelreyna/oneshot/v2

VERSION=`git describe --tags --abbrev=0`
VERSION_FLAG=$(GIT_REPO)/pkg/linkerSetValues.Version=$(VERSION)
STATIC_FLAG=-extldflags=-static
GOBIN=$(shell go env GOPATH)/bin

$(APP_NAME):
	mkdir -p ./build-output
	go build -o ./build-output/$(APP_NAME) \
		-ldflags "${STATIC_FLAG} -X ${VERSION_FLAG} -s -w" \
		./cmd/...

compressed: $(APP_NAME)
	upx --best --brute --no-lzma ./build-output/$(APP_NAME)

$(APP_NAME).1:
	go run -ldflags "-X ${VERSION_FLAG}" \
	       	./build-tools/man/main.go > ./build-output/$(APP_NAME).1

install-man-page: $(APP_NAME).1
	mv ./build-output/$(APP_NAME).1 $(MANPATH)/man1
	mandb

.PHONY: image
image:
	docker build -t $(IMAGE_REGISTRY)/$(APP_NAME):$(VERSION) .

.PHONY: itest
itest: $(APP_NAME)
	go test -count 1 -p 1 -timeout 30s ./integration_testing/...

.PHONY: lint
lint:
	$(GOBIN)/golangci-lint run
