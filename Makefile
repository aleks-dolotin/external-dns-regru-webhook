REPO_ROOT := $(shell pwd)
MOCK_LOG := /tmp/mock-regu.log
MOCK_PID := /tmp/mock-regu.pid

.PHONY: test test-verbose test-cover test-race lint vet build \
       mock-start mock-stop mock-status test-integration validate-openapi k8s-apply k8s-destroy

## ---------- Go build & quality ----------

build:
	@echo "Building sidecar..."
	go build -o bin/sidecar ./cmd/sidecar

test:
	@echo "Running Go unit tests..."
	go test ./...

test-verbose:
	@echo "Running Go unit tests (verbose)..."
	go test -v ./...

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

## ---------- Mock server & integration ----------

mock-start:
	@echo "Starting mock Reg.ru server..."
	@pkill -f 'tests/integration/mock-regru/server.py' || true
	@nohup python3 tests/integration/mock-regru/server.py > $(MOCK_LOG) 2>&1 & echo $$! > $(MOCK_PID)
	@sleep 1
	@echo "PID: $(shell cat $(MOCK_PID) 2>/dev/null || echo 'no pid')"
	@tail -n 20 $(MOCK_LOG) || true

mock-stop:
	@echo "Stopping mock Reg.ru server..."
	-@kill $$(cat $(MOCK_PID) 2>/dev/null) || true
	@rm -f $(MOCK_PID) || true

mock-status:
	@echo "Mock log: $(MOCK_LOG)"
	@tail -n 40 $(MOCK_LOG) || true

test-integration:
	@echo "Running integration tests (mock)"
	@python3 -m pytest -q tests/integration/mock-regru/test_mock_regu.py

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

