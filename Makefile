VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE ?= ghcr.io/aleks-dolotin/external-dns-regru-webhook

.PHONY: build test test-race test-smoke lint vet check clean docker-build docker-push

build:
	@echo "Building webhook (version=$(VERSION))..."
	go build -ldflags "-X main.Version=$(VERSION)" -o bin/webhook ./cmd/webhook

test:
	go test ./...

test-race:
	go test -race ./...

test-smoke:
	@echo "Running smoke tests against real Reg.ru API..."
	@echo "Required env: REGU_USERNAME, REGU_PASSWORD, SMOKE_TEST_ZONE"
	go test -v -tags=smoke -count=1 -timeout=300s ./tests/smoke/...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: vet test test-race
	@echo "All checks passed ✅"

clean:
	rm -rf bin/

docker-build:
	docker build -t $(IMAGE):$(VERSION) --build-arg VERSION=$(VERSION) .

docker-push: docker-build
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest
