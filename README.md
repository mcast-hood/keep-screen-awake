# keep-screen-awake

A lightweight background service that prevents your display and system from sleeping. Runs as a **Windows Service** (Windows) or **launchd agent** (macOS). Controlled via the `ksa` CLI.

Two complete implementations: **Go** and **C#/.NET 10**.

## Features

- Runs as a **Windows Service** or **macOS launchd agent** — starts automatically on login/boot
- Three modes: **always-on**, **toggle** (CLI on/off), or **schedule** (time windows)
- `ksa` CLI: `on`, `off`, `status`, `mode`, `logs`
- **IPC**: Named Pipe on Windows, HTTP REST on macOS
- `display_only` option — prevent only the display from sleeping, not the full system

---

## Modes

| Mode       | Behaviour |
|------------|-----------|
| `always`   | Sleep prevention is active the entire time the service runs |
| `toggle`   | Service runs always, but only prevents sleep when you run `ksa on` |
| `schedule` | Only prevents sleep during configured time windows |

---

## `display_only` explained

Controlled by the `display_only` field in `config.yaml`:

| Value | What is blocked |
|-------|----------------|
| `false` (default) | Display **and** system — screen stays on, machine never sleeps or hibernates |
| `true` | Display only — screen stays on, but the system itself may still sleep |

Use `false` for the typical "watching a video / running a process" case. Use `true` if you only need the screen on but are okay with the CPU going idle (e.g. laptop on battery).

---

## Repository Layout

```
keep-screen-awake/
├── go/                      Go implementation
│   ├── build.ps1            Windows build script (no make required)
│   ├── Makefile             macOS / Linux build
│   ├── cmd/
│   │   ├── ksad/            daemon binary
│   │   └── ksa/             CLI binary
│   └── internal/
│       ├── config/
│       ├── awake/           OS sleep prevention (Windows / macOS / stub)
│       └── transport/       Named Pipe (Windows) / HTTP (macOS)
├── dotnet/                  C#/.NET 10 implementation
│   ├── KeepScreenAwake.Core/    shared library
│   ├── KeepScreenAwake.Service/ daemon / Windows Service host
│   ├── KeepScreenAwake.Cli/     ksa CLI tool
│   └── KeepScreenAwake.Tests/   xUnit test suite
└── config.example.yaml      annotated configuration reference
```

---

## Prerequisites

### Go implementation

| Requirement | Notes |
|-------------|-------|
| Go 1.23+ | `go version` to check |
| `make` | **Optional** — macOS/Linux only. Windows uses `build.ps1` instead |
| Administrator shell | Required only for `ksad install / start / stop / uninstall` |

### C#/.NET implementation

| Requirement | Notes |
|-------------|-------|
| .NET 10 SDK | `dotnet --version` to check |
| Administrator shell | Required only for `sc.exe create / start / stop` |

---

## Configuration

Copy `config.example.yaml` to `config.yaml` and place it **next to the daemon binary**.

```yaml
# always | toggle | schedule
mode: always

schedule:
  - start: "09:00"
    end:   "18:00"
    days:  ["Mon", "Tue", "Wed", "Thu", "Fri"]

display_only: false

ipc:
  pipe_name: "keep-screen-awake"
  http_port:  9877

log:
  level: "info"
  file:  ""
```

---

## Go Implementation

### Build

**Windows** (no `make` needed):
```powershell
cd go
.\build.ps1            # produces bin\ksad.exe and bin\ksa.exe
```

**macOS / Linux**:
```bash
cd go
make build             # produces bin/ksad and bin/ksa
```

**macOS cross-compile from Windows**:
```powershell
.\build.ps1 -Darwin    # produces bin\darwin\ksad and bin\darwin\ksa
```

### Install & run (Windows)

> All `ksad` service commands require an **Administrator** PowerShell prompt.

```powershell
cd go

# 1. Build
.\build.ps1

# 2. Drop a config file next to the binary
Copy-Item ..\config.example.yaml bin\config.yaml
# Edit bin\config.yaml as needed (default mode: always is fine to start)

# 3. Install and start the Windows Service
bin\ksad.exe install
bin\ksad.exe start

# 4. Add ksa to your PATH (does NOT require Administrator)
#    Run this once, then open a new terminal
bin\ksad.exe add-to-path
```

After step 4, open a **new terminal** and:
```powershell
ksa status
```

### Install & run (macOS)

```bash
cd go

# 1. Build
make build

# 2. Drop a config file next to the binary
cp ../config.example.yaml bin/config.yaml

# 3. Install as a launchd user agent (starts on login)
./bin/ksad install

# 4. Start
./bin/ksad start

# 5. Add ksa to PATH (print the export line)
./bin/ksad add-to-path
```

### Service management

```powershell
ksad.exe start      # start the service
ksad.exe stop       # stop the service
ksad.exe uninstall  # remove the service
```

### Test

```powershell
cd go
.\build.ps1 -Test
# or on macOS: make test
```

---

## C#/.NET 10 Implementation

### Build

```bash
cd dotnet
dotnet build -c Release
```

### Publish self-contained (Windows)

```powershell
dotnet publish KeepScreenAwake.Service/KeepScreenAwake.Service.csproj `
  -c Release -r win-x64 --self-contained -o publish\service

dotnet publish KeepScreenAwake.Cli/KeepScreenAwake.Cli.csproj `
  -c Release -r win-x64 --self-contained -o publish\cli
```

### Install & run (Windows)

```powershell
# Copy config next to the service binary
Copy-Item ..\config.example.yaml publish\service\config.yaml

# Install as a Windows Service (Administrator required)
sc.exe create KeepScreenAwake binPath="$PWD\publish\service\KeepScreenAwake.Service.exe"
sc.exe description KeepScreenAwake "Prevents display and system sleep."
sc.exe start KeepScreenAwake

# Add ksa CLI to PATH
$ksaDir = "$PWD\publish\cli"
[Environment]::SetEnvironmentVariable('PATH', $env:PATH + ";$ksaDir", 'User')
# Open a new terminal, then: ksa status
```

### Test

```bash
cd dotnet
dotnet test
```

---

## CLI (`ksa`)

```bash
# Show current status (mode, awake active, display_only)
ksa status

# Enable sleep prevention (useful in toggle mode)
ksa on

# Disable sleep prevention
ksa off

# Switch mode at runtime
ksa mode always
ksa mode toggle
ksa mode schedule

# Show recent log lines from the daemon
ksa logs
ksa logs --lines 100
```

---

## IPC Protocol

### Windows — Named Pipe

Pipe: `\\.\pipe\keep-screen-awake`

Each connection carries one newline-delimited JSON request and one JSON response.

```json
{"command":"status"}
{"command":"on"}
{"command":"off"}
{"command":"mode","mode":"schedule"}
{"command":"logs","lines":50}
```

### macOS — REST HTTP

Base URL: `http://127.0.0.1:9877`

| Method | Path     | Notes |
|--------|----------|-------|
| POST   | /command | JSON `Request` body |
| GET    | /status  | Convenience status check |

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│  ksad  (daemon)                                 │
│                                                 │
│  ┌───────────────┐  ┌──────────────────────┐   │
│  │  AwakeManager │  │     IPC Server       │   │
│  │  Win/macOS    │  │  Pipe (Win) / HTTP   │   │
│  └───────┬───────┘  └──────────┬───────────┘   │
│          └──────── Mode Engine ┘                │
│             always / toggle / schedule          │
└─────────────────────────────────────────────────┘
              ▲ Named Pipe / HTTP ▲
┌─────────────┴──────────────────┴──────────────┐
│  ksa  (CLI)                                   │
│  status  │  on  │  off  │  mode  │  logs      │
└────────────────────────────────────────────────┘
```

---

## Troubleshooting

### `ksa` is not recognized / CommandNotFoundException

`ksa.exe` is not on your `PATH`. Fix:

```powershell
# Permanently add to user PATH (no Administrator required)
bin\ksad.exe add-to-path
```

Then **open a new terminal**. The change only applies to new shells.

Alternatively, add the path manually:
```powershell
[Environment]::SetEnvironmentVariable(
    'PATH',
    [Environment]::GetEnvironmentVariable('PATH','User') + ';C:\Repos\keep-screen-awake\go\bin',
    'User'
)
```

---

### `make: command not found` on Windows

Windows does not ship with `make`. Use the PowerShell build script instead:

```powershell
cd go
.\build.ps1          # build
.\build.ps1 -Test    # run tests
.\build.ps1 -Clean   # remove bin\
```

---

### Service fails to start / exits immediately

1. Check that `config.yaml` exists **next to `ksad.exe`** (in the `bin\` folder).
2. Check the Windows Event Viewer → Windows Logs → Application for errors from `KeepScreenAwake`.
3. Test in the foreground first (no Administrator needed):
   ```powershell
   bin\ksad.exe run
   ```

---

### `ksa status` hangs and never returns

This was a bug in early builds where the named pipe used `io.ReadAll` (blocking until EOF) instead of newline-delimited reads. It is fixed in the current version. If you have an old binary, rebuild:

```powershell
cd go
.\build.ps1
bin\ksad.exe stop
bin\ksad.exe start
```
