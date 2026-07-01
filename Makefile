SHELL := /usr/bin/env bash
SCRIPTS := bin/c48x-airbridge scripts/common.sh scripts/diagnose.sh scripts/install-cups.sh scripts/install-sane-samsung.sh scripts/install-airsane.sh scripts/bootstrap-setup.sh

.PHONY: check syntax dry-run setup setup-dry-run help

help:
	@printf 'Targets:\n  check          Run syntax and bootstrap dry-run checks\n  syntax         Run bash -n over scripts\n  dry-run        Alias for setup-dry-run\n  setup-dry-run  Preview bootstrap/build path without mutating host\n  setup          Run guided setup bootstrap\n'

check: syntax setup-dry-run

syntax:
	@for script in $(SCRIPTS); do bash -n "$$script"; done

dry-run:
	@$(MAKE) setup-dry-run

setup-dry-run:
	@./scripts/bootstrap-setup.sh --dry-run --no-input >/dev/null

setup:
	@./scripts/bootstrap-setup.sh
