# JSON report fields

Live and saved-report modes use the same ProcSift JSON parser. Released 0.5.x reports are supported. The parser also accepts the current unreleased `0.6.0-dev` report for testing; compatibility with a final 0.6 release has not been established.

| View | Fields read from the report |
| --- | --- |
| Header | `version`, `time`, `host.hostname`, `host.boot_id`, `host_result`, `host_counts`, `scanner_counts` |
| Findings | `findings[]`: severity, rule, PID or subject, summary, detail, evidence, origin |
| Process detail | `processes[]`: PID, parent, UID, state, name, executable, command line, start ticks, image identity and hash |
| Related sockets | `sockets[]`, joined by recorded PID |
| Coverage | `coverage[]`, `limits[]`, `notes[]` |
| Scanner self | Findings with `origin: scanner_self` |

The process list is compact. It does not include every memory map, file descriptor, service, or startup reference collected by ProcSift. A missing field appears as unknown in the UI. Socket ownership and collection coverage can also be incomplete.

The parser rejects reports over 64 MiB, malformed JSON, capture manifests, and reports with an unsupported tool or version. The folder picker displays at most 1,000 regular `.json` files from one directory; it does not search subdirectories or follow file symlinks.
