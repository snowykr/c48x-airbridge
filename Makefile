SHELL := /usr/bin/env bash
SCRIPTS := bin/local-printer-scanner scripts/common.sh scripts/diagnose.sh scripts/install-cups.sh scripts/install-sane-samsung.sh scripts/install-airsane.sh

.PHONY: check syntax dry-run help

help:
	@printf 'Targets:\n  check      Run syntax and dry-run checks\n  syntax     Run bash -n over scripts\n  dry-run    Print installer commands without mutating host\n'

check: syntax dry-run

syntax:
	@for script in $(SCRIPTS); do bash -n "$$script"; done

dry-run:
	@./bin/local-printer-scanner install --dry-run >/dev/null
