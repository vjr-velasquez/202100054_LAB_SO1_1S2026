PYTHON ?= python3
COMPOSE = docker compose -f deploy/compose.local.yaml

.PHONY: check check-reference check-services doctor build local-up local-down check-integration check-cluster phase4-up check-charts

check: check-reference check-services check-charts

check-reference:
	$(PYTHON) scripts/check.py
	$(PYTHON) -B -m unittest discover -s tests -p 'test_*.py' -v

check-services: build

build:
	sh scripts/build.sh

doctor:
	$(PYTHON) scripts/doctor.py

local-up: build
	$(COMPOSE) up -d --no-build --wait

local-down:
	$(COMPOSE) down

check-integration: local-up
	$(COMPOSE) run --rm go-tests
	$(PYTHON) scripts/integration.py

phase4-up:
	$(PYTHON) scripts/phase4-up.py

check-charts:
	$(PYTHON) scripts/k8s-tools.py
	artifacts/tools/helm lint operator/helm-charts/valkeyinstance
	artifacts/tools/helm lint operator/helm-charts/grafanainstance
	artifacts/tools/kubectl kustomize operator/config/default > /dev/null

check-cluster:
	$(PYTHON) scripts/check-cluster.py
