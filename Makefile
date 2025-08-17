# Makefile

# Use an overridable Python interpreter (defaults to python)
PYTHON ?= python

.PHONY: configure copy-conf-files
configure:
	@echo "Running configure.py..."
	$(PYTHON) -m app.configure
