REPO_ROOT := $(shell pwd)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: test test-cover test-race lint vet build \
       setup-hooks pre-commit \
       mock-start mock-stop mock-status test-integration test-integration-python test-smoke validate-openapi k8s-apply k8s-destroy

## ---------- Go build & quality ----------

build:
	@echo "Building sidecar (version=$(VERSION))..."
	go build -ldflags "-X main.Version=$(VERSION)" -o bin/sidecar ./cmd/sidecar

test:
	@echo "Running Go unit tests..."
	go test ./...


test-race:
	@echo "Running Go unit tests with race detector..."
	go test -race ./...

test-cover:
	@echo "Running Go unit tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "HTML report: go tool cover -html=coverage.out"

vet:
	@echo "Running go vet..."
	go vet ./...

lint:
	@echo "Running golangci-lint..."
	@which golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found — install: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run ./...

check: vet test test-race
	@echo "All checks passed ✅"

## ---------- Git hooks ----------

setup-hooks:
	@echo "Installing pre-commit hook..."
	@cp scripts/pre-commit.sh .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed ✅"

pre-commit:
	@echo "Running pre-commit checks manually..."
	@bash scripts/pre-commit.sh

## ---------- Mock server & integration ----------

mock-start:
	@echo "Starting mock Reg.ru server..."
	@bash devtools/mock-regru/start_mock.sh

mock-stop:
	@echo "Stopping mock Reg.ru server..."
	@bash devtools/mock-regru/stop_mock.sh

mock-status:
	@echo "Mock log: /tmp/mock-regru.log"
	@tail -n 40 /tmp/mock-regru.log || true

test-integration: mock-start
	@echo "Running Go integration tests..."
	REGRU_BASE_URL=http://127.0.0.1:8081/api/regru2 go test -v -tags=integration ./tests/integration/... || ($(MAKE) mock-stop && exit 1)
	@$(MAKE) mock-stop
	@echo "Integration tests passed ✅"

test-integration-python:
	@echo "Running Python integration tests (mock-regru)..."
	@python3 -m pytest -q tests/integration/mock-regru/test_mock_regu.py

test-smoke:
	@echo "Running smoke tests against real Reg.ru API..."
	@echo "Required env: REGU_USERNAME, REGU_PASSWORD, SMOKE_TEST_ZONE"
	go test -v -tags=smoke -count=1 -timeout=300s ./tests/smoke/...

validate-openapi:
	@node -v >/dev/null 2>&1 || echo "Node not found — install node to run openapi-cli"
	@npm i -g @redocly/openapi-cli >/dev/null 2>&1 || true
	@openapi validate docs/openapi/adapter-v1.yaml || true

k8s-apply:
	@echo "Apply k8s manifests (namespace, configmap, secret template, deployments, hpa, pdb, service)"
	@kubectl apply -f k8s/deploy/namespace.yaml
	@kubectl apply -f k8s/deploy/configmap-reg-adapter.yaml
	@kubectl apply -f k8s/deploy/secret-reg-credentials.yaml || true
	@kubectl apply -f k8s/deploy/deployment-reg-adapter.yaml
	@kubectl apply -f k8s/deploy/deployment-reg-worker.yaml
	@kubectl apply -f k8s/deploy/service-and-monitor.yaml || true
	@kubectl apply -f k8s/deploy/hpa-reg-worker.yaml || true
	@kubectl apply -f k8s/deploy/pdb-reg-adapter.yaml || true

k8s-destroy:
	@echo "Delete k8s resources"
	@kubectl delete -f k8s/deploy/pdb-reg-adapter.yaml --ignore-not-found
	@kubectl delete -f k8s/deploy/hpa-reg-worker.yaml --ignore-not-found
	@kubectl delete -f k8s/deploy/service-and-monitor.yaml --ignore-not-found
	@kubectl delete -f k8s/deploy/deployment-reg-worker.yaml --ignore-not-found
	@kubectl delete -f k8s/deploy/deployment-reg-adapter.yaml --ignore-not-found
	@kubectl delete -f k8s/deploy/secret-reg-credentials.yaml --ignore-not-found
	@kubectl delete -f k8s/deploy/configmap-reg-adapter.yaml --ignore-not-found
	@kubectl delete -f k8s/deploy/namespace.yaml --ignore-not-found

