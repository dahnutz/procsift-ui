# ProcSift UI design

ProcSift UI is a terminal viewer for ProcSift JSON reports. A live session runs the scanner once and displays its report. Saved reports open from a file or a directory picker. Navigation stays within the loaded snapshot.

For commands and keys, see [README.md](README.md). [REPORT_FORMAT.md](REPORT_FORMAT.md) lists the report fields used by the viewer.

Copyright © 2026 Daniel Hanganu. [MIT licensed](LICENSE).

## Input

Live mode runs `procsift -json -q` without a shell. The scanner is found on `PATH` or supplied with `--scanner`. Exit `0` means no finding crossed ProcSift's configured threshold; exit `1` can still carry a valid report. A scan failure or invalid JSON stops the UI before it changes terminal mode.

`--file` reads one regular JSON report. `--folder` lists regular `.json` files in one directory and opens the selected report. The picker does not recurse or follow file symlinks. Both modes use saved data and make no new scan. JSON input is capped at 64 MiB.

## Views

The first list shows host findings from high to info. ProcSift's Attention score orders findings within a severity where a score exists; the full finding list remains available. Separate tabs show processes, collection coverage, and findings produced by the scanner itself.

A finding opens its recorded summary, detail, evidence, and related PID. A PID view uses the process, sockets, and findings in that same report. The UI shows the report time and file name so saved data remains identifiable as a snapshot. It does not read more process data when a detail view opens.

## Terminal handling

The viewer needs a real terminal. It uses an alternate screen and restores terminal mode, cursor, and display state on exit. Report text is sanitized before display so control characters in process names, paths, command lines, or evidence cannot act as terminal commands.

The executable does not acquire memory, execute files found in a report, or contact recorded network endpoints. The scanner and viewer are separate binaries connected through ProcSift's JSON output. The UI code and its terminal dependency add no code to the scanner binary.

## Build and limits

`make build` produces `bin/procsift-ui` as a static Linux executable. `make test` checks parsing, sorting, saved reports, scanner exit handling, terminal rendering, and navigation. `make verify-binary` rebuilds with the current toolchain and compares the bytes. The UI's compact process view is limited to fields present in the JSON report; missing socket ownership or collection data remains unknown.
