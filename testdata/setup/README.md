# Setup fake-runner fixtures

`setup --fake-runner` accepts JSON fixtures under `testdata/setup/` only. The
fixture runs inside `C48X_AIRBRIDGE_FAKE_ROOT`; tests must never write real
`/etc`, `/var`, or other host paths.

Schema fields:

- `name`: fixture name for output.
- `command_log_path`: absolute path, rooted under the fake root at apply time.
- `file_preloads`: files to create before apply. Each item has `path`, `content`,
  and optional numeric `mode`.
- `steps`: ordered runner steps. A `command` step has `name`, `command`, optional
  `args`, and optional `sudo`. A `write_file` step has `path`, `content`, and
  optional numeric `mode`.
- `commands`: expected command results keyed by command line without `sudo`.
  Each result has `exit_code`, optional `stdout`, and optional `stderr`.
- `probe_outputs`: reserved fixture probe outputs for later setup phases.
- `expected_writes`: file paths and expected content after apply.

Command failures produce setup state `FAIL` and include rollback and retry
guidance plus the command log path in command output.
