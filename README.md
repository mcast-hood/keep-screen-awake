# keep-screen-awake

A lightweight background service that prevents your display and system from sleeping. Runs as a **Windows Service** (Windows) or **launchd agent** (macOS). Controlled via the `ksa` CLI.

Two complete implementations: **Go** and **C#/.NET 10**.

## Features

- Runs as a **Windows Service** or **macOS launchd agent** — starts automatically on login
- Three modes: **always-on**, **toggle** (CLI on/off), or **schedule** (time windows)
- `ksa` CLI: `on`, `off`, `status`, `mode`, `logs`
- **IPC**: Named Pipe on Windows, HTTP REST on macOS
- `display_only` option — prevent only the display from sleeping, not the full system

---

## Modes

| Mode       | Behaviour                                                      |
|------------|----------------------------------------------------------------|
| `always`   | Sleep prevention is active the entire time the service runs   |
| `toggle`   | Service runs always, but only prevents sleep when you run `ksa on` |
| `schedule` | Only prevents sleep during configured time windows            |

---

## Repository Layout

```
keep-screen-awake/
├── go/                      Go implementation
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

## Configuration

Copy `config.example.yaml` to `config.yaml` and place it next to the daemon binary.

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

### Requirements

- Go 1.23+

### Build

```bash
cd go

# Windows binaries
make build

# macOS binaries
make build-darwin
```

Output lands in `go/bin/`.

### Install & run (Windows)

```powershell
# Add bin to PATH (once, then reopen terminal)
[Environment]::SetEnvironmentVariable(
    'PATH',
    [Environment]::GetEnvironmentVariable('PATH','User') + ';' + (Resolve-Path .\bin),
    'User'
)

# Install as a Windows Service (run as Administrator)
.\bin\ksad.exe install

# Start / stop
.\bin\ksad.exe start
.\bin\ksad.exe stop

# Remove
.\bin\ksad.exe uninstall
```

### Install & run (macOS)

```bash
# Add bin to PATH
echo 'export PATH="$PATH:'"$(pwd)/bin"'"' >> ~/.zshrc

# Install as a launchd user agent
./bin/ksad-darwin install

# Start / stop
./bin/ksad-darwin start
./bin/ksad-darwin stop

# Remove
./bin/ksad-darwin uninstall

# Run in the foreground for debugging
./bin/ksad-darwin run
```

### Build (test only)

```bash
cd go
make test
```

---

## C#/.NET 10 Implementation

### Requirements

- .NET 10 SDK

### Build

```bash
cd dotnet
dotnet build -c Release
```

### Publish self-contained (Windows)

```bash
dotnet publish KeepScreenAwake.Service/KeepScreenAwake.Service.csproj \
  -c Release -r win-x64 --self-contained -o publish/service

dotnet publish KeepScreenAwake.Cli/KeepScreenAwake.Cli.csproj \
  -c Release -r win-x64 --self-contained -o publish/cli
```

### Install & run (Windows)

```powershell
# Install as a Windows Service (run as Administrator)
sc.exe create KeepScreenAwake binPath="C:\path\to\publish\service\KeepScreenAwake.Service.exe"
sc.exe description KeepScreenAwake "Prevents display and system sleep."
sc.exe start KeepScreenAwake
```

### Install & run (macOS)

Run `KeepScreenAwake.Service` as a foreground process or wrap it in a launchd plist.

### Test

```bash
cd dotnet
dotnet test
```

---

## CLI (`ksa`)

```bash
# Show current status (mode, awake active, next schedule window)
ksa status

# Enable sleep prevention (toggle mode)
ksa on

# Disable sleep prevention (toggle mode)
ksa off

# Switch mode at runtime
ksa mode always
ksa mode toggle
ksa mode schedule

# Show recent log lines
ksa logs
ksa logs --lines 100
```

---

## IPC Protocol

### Windows — Named Pipe

Pipe: `\\.\pipe\keep-screen-awake`

Each connection: one newline-delimited JSON request → one JSON response.

```json
// Requests
{"command":"status"}
{"command":"on"}
{"command":"off"}
{"command":"mode","mode":"schedule"}
{"command":"logs","lines":50}

// Response
{"ok":true,"data":{...}}
{"ok":false,"error":"unknown command"}
```

### macOS — REST HTTP

Base URL: `http://127.0.0.1:9877`

| Method | Path       | Notes                   |
|--------|------------|-------------------------|
| POST   | /command   | JSON `Request` body     |
| GET    | /status    | Convenience status check|

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│  ksad  (daemon)                                 │
│                                                 │
│  ┌───────────────┐  ┌──────────────────────┐   │
│  │  AwakeManager │  │     IPC Server       │   │
│  │  (Win/macOS)  │  │  Pipe / HTTP         │   │
│  └───────┬───────┘  └──────────┬───────────┘   │
│          │                     │               │
│          └─────────────────────┘               │
│                  Mode Engine                    │
│           always / toggle / schedule            │
└─────────────────────────────────────────────────┘
              ▲ Named Pipe / HTTP ▲
┌─────────────┴──────────────────┴──────────────┐
│  ksa  (CLI)                                   │
│  status  │  on  │  off  │  mode  │  logs      │
└────────────────────────────────────────────────┘
```
