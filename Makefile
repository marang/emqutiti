PROTO_FILES := proxy/proxy.proto
PROTOC ?= protoc
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/marang/emqutiti/cmd.version=$(VERSION)

.PHONY: build test vet proto tape

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o emqutiti ./cmd/emqutiti

vet:
	go vet ./...

test: vet
	go test ./...

proto:
	$(PROTOC) --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. $(PROTO_FILES)

tape:
	docker build -f docs/scripts/Dockerfile.vhs -t emqutiti-tape docs/scripts
	docker run --rm -it \
		--user $(shell id -u):$(shell id -g) \
		-e HOME=/tmp \
		--security-opt seccomp=unconfined \
		--security-opt apparmor=unconfined \
		--cap-add=SYS_ADMIN \
		-v "$(CURDIR)":/work -w /work \
		emqutiti-tape docs/scripts/record_tapes.sh
