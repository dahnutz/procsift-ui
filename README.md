# ProcSift UI

Terminal viewer for [ProcSift](https://github.com/dahnutz/procsift) JSON reports. It can run a live scan through the separate `procsift` executable, open one saved report, or let you choose a report from a folder. The UI reads the report; ProcSift does the scanning.

## Download and run

For a live scan on Linux x86-64, download both executables and run the UI:

```sh
wget -O procsift https://github.com/dahnutz/procsift/raw/refs/heads/main/bin/procsift
wget -O procsift-ui https://github.com/dahnutz/procsift-ui/raw/refs/heads/main/bin/procsift-ui
chmod +x procsift procsift-ui
sudo ./procsift-ui --scanner ./procsift
```

For an existing JSON report, run `./procsift-ui --file report.json` instead. Use the same account that can read the report.

## Build

Go 1.23 or newer is needed to build from source:

```sh
make build
./bin/procsift-ui --help
```

The executable is [bin/procsift-ui](bin/procsift-ui). `make test` runs the tests, `make verify-binary` compares it with a fresh build, and `make size` shows its size. The UI is a separate Go module and does not import ProcSift's scanner code.

## Run a live scan

If `procsift` is on `PATH`:

```sh
sudo ./bin/procsift-ui
```

Otherwise, point the UI at the scanner executable. From this directory, with the ProcSift source checkout next to it:

```sh
sudo ./bin/procsift-ui --scanner ../procsift/bin/procsift
```

The UI runs `procsift -json -q` and opens that snapshot. It accepts scanner exit `1`, which means a finding met ProcSift's threshold. Press `r` on the main screen to scan again. Opening a finding or PID does not start another scan.

Press `s` on the main screen and enter a file name or path. Use `.txt` for a readable summary of the findings, processes, sockets, file checks, and coverage. Use `.json` for the complete scanner report; a name without an extension gets `.json` automatically. Press Enter on an empty prompt to use the suggested JSON name. The suggested name changes when that file already exists. Left and right arrows move through a typed name; Backspace and Delete edit it. The UI creates files with mode `0600` and will not overwrite an existing file. A failed refresh leaves the previous snapshot on screen.

## Open a saved report

```sh
sudo ../procsift/bin/procsift -json -o /tmp/procsift-first.json
sudo ./bin/procsift-ui --file /tmp/procsift-first.json
```

Use `--folder /path/to/reports` to choose among `.json` files in one directory. These modes do not start the scanner. ProcSift creates report files at mode `0600`, so open them as the same user that ran the scan. The folder picker does not follow symlinks or search subdirectories.

## Keys

| Key | Action |
| --- | --- |
| Up/Down or `j`/`k` | Move through a list |
| Page Up/Page Down | Move a page |
| Enter | Open a finding, process, or selected report |
| `p` | Enter a PID recorded in this report |
| `n` / `p` in a detail view | Open the next or previous finding or process in the current list |
| `/` | Filter the current list |
| `r` on the live main screen | Run a new scan |
| `s` on the live main screen | Save the current scan as JSON or text |
| Tab | Switch between Findings, Processes, Coverage, and Scanner self |
| Esc | Return to the list or cancel a prompt |
| `q` | Quit |

Findings are ordered high to low. In a detail view, `n` and `p` follow the current list order, including any filter. A PID opened with `p` on the main screen follows numeric PID order. The detail view shows the evidence recorded in the report. PID details include its recorded process fields, sockets, and linked findings. Coverage lists incomplete reads and scan limits. The report time and source remain visible so an offline snapshot is not mistaken for live state.

The UI needs an interactive terminal. It reads released ProcSift JSON reports up to 64 MiB. Process details are limited to fields already present in a report; the UI does not read `/proc` or acquire more evidence. [REPORT_FORMAT.md](REPORT_FORMAT.md) lists the fields it uses, and [PROPOSAL.md](PROPOSAL.md) describes the design.

Copyright © 2026 Daniel Hanganu. [MIT licensed](LICENSE).
