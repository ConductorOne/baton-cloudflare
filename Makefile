GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
BUILD_DIR = dist/${GOOS}_${GOARCH}
OUTPUT_PATH = ${BUILD_DIR}/baton-cloudflare
GENERATED_CONF = pkg/config/conf.gen.go

.PHONY: generate
generate: $(GENERATED_CONF)

$(GENERATED_CONF): pkg/config/config.go go.mod
	go generate ./pkg/config

.PHONY: build
build: $(GENERATED_CONF)
	rm -f ${OUTPUT_PATH}
	mkdir -p ${BUILD_DIR}
	go build -o ${OUTPUT_PATH} cmd/baton-cloudflare/*.go

.PHONY: update-deps
update-deps:
	go get -d -u ./...
	go mod tidy -v
	go mod vendor

.PHONY: add-dep
add-dep:
	go mod tidy -v
	go mod vendor

.PHONY: lint
lint:
	golangci-lint run
