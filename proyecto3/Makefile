PYTHON ?= python3
COMPOSE = docker compose -f deploy/compose.local.yaml

.PHONY: check check-reference check-services doctor build local-up local-down check-integration check-cluster

check: check-reference check-services

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

check-cluster:
	@echo 'PENDIENTE: Operator y GKE se verifican en las fases 4 y 5.' >&2
	@exit 2
