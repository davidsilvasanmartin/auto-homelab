# Makefile

# Use an overridable Python interpreter (defaults to python)
PYTHON ?= python
DOCKER ?= docker

.PHONY: configure up
configure:
	@echo "Running configure.py..."
	$(PYTHON) -m app.configure

up:
	@echo "Enabling all services..."
	$(DOCKER) compose up -d
