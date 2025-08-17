# Makefile

# Use an overridable Python interpreter (defaults to python)
PYTHON ?= python

.PHONY: configure copy-conf-files
configure:
	@echo "Running configure.py..."
	$(PYTHON) -m app.configure

copy-conf-files:
	@echo "Copying configuration files..."
	$(PYTHON) -m app.copy_conf_files
