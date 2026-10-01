PYTHON ?= python3

.PHONY: check doctor check-integration check-cluster

check:
	$(PYTHON) scripts/check.py
	$(PYTHON) -B -m unittest discover -s tests -p 'test_*.py' -v

doctor:
	$(PYTHON) scripts/doctor.py

check-integration:
	@echo 'PENDIENTE: servicios e integración se implementan en las fases 3 y 4.' >&2
	@exit 2

check-cluster:
	@echo 'PENDIENTE: Operator y GKE se verifican en las fases 4 y 5.' >&2
	@exit 2
