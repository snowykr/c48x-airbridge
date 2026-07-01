SHELL := /usr/bin/env bash
SCRIPTS := bin/c48x-airbridge scripts/common.sh scripts/diagnose.sh scripts/install-cups.sh scripts/install-sane-samsung.sh scripts/install-airsane.sh scripts/bootstrap-setup.sh

.PHONY: check syntax dry-run setup help

help:
	@printf 'Targets:\n  check      Run syntax and dry-run checks\n  syntax     Run bash -n over scripts\n  dry-run    Print installer commands without mutating host\n  setup      Run guided setup bootstrap\n'

check: syntax dry-run

syntax:
	@for script in $(SCRIPTS); do bash -n "$$script"; done

dry-run:
	@./bin/c48x-airbridge install --dry-run >/dev/null

setup:
	@./scripts/bootstrap-setup.sh
