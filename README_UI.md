# DCS Desktop GUI (`dcs-gui`)

Fyne-based desktop front-end for DCS. It lives in `cmd/gui` and calls the same
`internal/*` packages as the CLI, so behaviour matches the command-line tool.

---

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| **Go 1.24+** | See `go.mod` in the repo root. |
| **C toolchain** | Fyne needs CGO. Install platform-specific tools below. |
| **`CGO_ENABLED=1`** | **Required when building or running the GUI.** Without it, `go build` / `go run` for `./cmd/gui` will fail. |
| **`.env` file** | Place in the directory you launch the app from (see [Environment](#environment)). |

### C toolchain by platform

**Windows** — install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or mingw-w64:

```powershell
winget install -e --id BrechtSanders.WinLibs.POSIX.UCRT
```

Ensure `gcc.exe` is on `PATH`.

**Linux** (Debian/Ubuntu):

```bash
sudo apt install gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev
```

**macOS** — Xcode Command Line Tools:

```bash
xcode-select --install
```

---

## First-time setup

From the repository root:

```bash
go mod tidy
```

---

## Run (no build step)

From the repository root, with `.env` in place:

```bash
# Linux / macOS
CGO_ENABLED=1 go run ./cmd/gui
```

```powershell
# Windows
$env:CGO_ENABLED = "1"
go run ./cmd/gui
```

`go run` compiles and launches the GUI in one step — no `dcs-gui` binary is written to disk. You still need `CGO_ENABLED=1` and the C toolchain (same as a full build).

---

## Build a binary

Set `CGO_ENABLED=1` when producing a standalone executable.

Unlike the CLI (`CGO_ENABLED=0` cross-builds easily with `GOOS` / `GOARCH`), the GUI uses **Fyne + CGO**. The reliable approach is to **compile on the same OS (and usually the same CPU architecture) as the machine that will run the app**. Copy the resulting binary plus your `.env` (and `accounts/` if needed) to the target PC; Go is not required on the machine that only runs `dcs-gui`.

### On the machine that will run the GUI (recommended)

On each target, install [Go](#prerequisites), the [C toolchain](#c-toolchain-by-platform) for that OS, then from the repo root:

**Linux**

```bash
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o dcs-gui ./cmd/gui
./dcs-gui
```

**macOS**

```bash
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o dcs-gui ./cmd/gui
./dcs-gui
```

On Apple Silicon, Go defaults to `GOARCH=arm64`. On Intel Macs, `GOARCH=amd64` is the default — no extra flags unless you know you need them.

**Windows (PowerShell)**

```powershell
$env:CGO_ENABLED = "1"
go build -trimpath -ldflags "-s -w" -o dcs-gui.exe ./cmd/gui
.\dcs-gui.exe
```

### CPU architecture on the same OS

If the build machine matches the target OS but not the CPU (for example building on Linux x86_64 for Linux ARM64), set `GOARCH` and use a C compiler for that architecture (often a prefixed cross-gcc, e.g. `aarch64-linux-gnu-gcc`):

```bash
# Example: Linux amd64 host → Linux arm64 binary
CGO_ENABLED=1 GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
  go build -trimpath -ldflags "-s -w" -o dcs-gui ./cmd/gui
```

If `CC` is wrong or missing, the link step fails — that usually means you should build natively on the target device instead.

### Building on one machine for another OS

Cross-compiling the GUI from Windows to Linux (or the reverse) is **not** the same as the CLI one-liners in [README.md](README.md): CGO needs a C toolchain and system libraries for the **destination** platform.

| Approach | When to use |
|----------|-------------|
| **Build on the target** | Simplest and most reliable for demos and releases. |
| **CI per OS** | e.g. GitHub Actions with `windows-latest`, `ubuntu-latest`, `macos-latest` — each job builds on its own runner (same pattern as this repo’s CI). |
| **VM / second machine** | Clone the repo on a Linux laptop, Windows PC, or Mac and build there. |
| **Cross-compiler on the host** | Advanced: install mingw-w64 (Linux→Windows), `x86_64-linux-gnu-gcc` (Windows→Linux via MSYS2/WSL), etc., set `GOOS`, `GOARCH`, and `CC`. Easy to get wrong; prefer native builds unless you already maintain cross toolchains. |
| **[fyne-cross](https://github.com/fyne-io/fyne-cross)** | Docker-based Fyne builds for multiple targets from one host (requires Docker). |

There is no supported `CGO_ENABLED=0` GUI build: a binary built that way will not link Fyne.

### Deploying to another machine

1. Build on the target OS (or copy a binary built on a matching OS/arch).
2. Copy `dcs-gui` / `dcs-gui.exe` to the machine.
3. Place `.env` (from `.env.example`) in the folder you launch from.
4. Install **IPFS (Kubo)** on that machine if you use User IPFS workflows (or set `IPFS_BIN` in `.env`).
5. Run the binary from that folder — no `go run` and no repo checkout required on the target.

`go run ./cmd/gui` is for development on a machine that already has Go and the C toolchain; it is not how you ship to another computer.

### WSL and remote Linux

- **WSL2**: treat as Linux — install the Linux C deps and build inside WSL. You need a display server (WSLg on Windows 11, or X11 forwarding) to see the window.
- **SSH-only server**: no display — the GUI cannot run there; build on a desktop Linux machine or use the CLI instead.

---

## Environment

### `.env` file (runtime)

On startup the GUI calls `config.LoadEnv()`, which reads **`.env` from the current working directory** (the folder you run the binary from, not necessarily the repo root).

Create it from the example at the repo root:

```powershell
# Windows
copy .env.example .env
```

```bash
# Linux / macOS
cp .env.example .env
```

Edit `.env` with your testnet identities and DCS object IDs. Values pre-fill the
Ground Control, User, and Provider identity panels; you can override any field
in the UI.

### Variables used by the GUI

| Variable | Used for |
|----------|----------|
| `REBASE_RPC` / `DCS_RPC` | IOTA Rebased RPC URL (default: testnet) |
| `IOTA_GRAPHQL_ENDPOINT` | GraphQL endpoint for reads |
| `GC_ENDPOINT` | Ground Control HTTP API (GC actor) |
| `GC_API_TOKEN` | GC API auth token (not in `.env.example`; add if you use GC views) |
| `USER_PRIVATE_KEY`, `USER_ADDRESS`, `USER_GAS_COIN_ID` | User actor defaults |
| `PROVIDER_PRIVATE_KEY`, `PROVIDER_ADDRESS`, `PROVIDER_GAS_COIN_ID` | Provider actor defaults |
| `ACTIVE_PRIVATE_KEY`, `ACTIVE_ADDRESS`, `ACTIVE_GAS_COIN_ID` | Override active wallet (User) |
| `DCS_PACKAGE_ID`, `DCS_WHITELIST_ID`, `DCS_CIDLIST_ID`, `DCS_CLOCK_ID` | On-chain DCS objects |
| `WALLET_GAS_BUDGET` | Transaction gas budget |
| `FAUCET_URL` | Faucet for new accounts created in the GUI |
| `IPFS_BIN` | Optional full path to `ipfs` if it is not on `PATH` |

See `.env.example` at the repo root for sample values and formats.

### Build-time environment

| Variable | When |
|----------|------|
| `CGO_ENABLED=1` | **Always** when building or `go run`-ning `./cmd/gui` |

You can set it once instead of prefixing every command:

**Windows (persistent user variable)** — System Settings → Environment Variables, or in PowerShell for the current user:

```powershell
[Environment]::SetEnvironmentVariable("CGO_ENABLED", "1", "User")
```

Restart the terminal after changing user/system variables.

**Go toolchain (persistent for your user)**:

```bash
go env -w CGO_ENABLED=1
```

Stored in Go’s env file (e.g. `%USERPROFILE%\AppData\Roaming\go\env` on Windows). Override per command when needed (`CGO_ENABLED=0 go build ./cmd/cli` still works for the CGO-free CLI).

**Not in project `.env`** — DCS loads `.env` only when the GUI (or CLI) **runs**, via `config.LoadEnv()`. `go build` and `go run` do not read that file, so `CGO_ENABLED` there has no effect on compiling the GUI. Use OS env, shell profile, or `go env -w` instead.

### Optional runtime fixes

- **`LANG` / `LC_ALL`** — On WSL or minimal shells where `LANG=C`, Fyne may log locale warnings. The app sets `LANG=en_US.UTF-8` automatically when needed.
- **`IPFS_BIN`** — Set if the GUI cannot find `ipfs` on `PATH` when starting the embedded IPFS daemon from Settings.

---

## Working directory

Run `dcs-gui` from a directory that contains:

1. **`.env`** — loaded at startup.
2. **`accounts/`** (optional) — wallet files created by the CLI or GUI (`accounts/<alias>.json`).

Example from the repo root after configuring `.env`:

```bash
CGO_ENABLED=1 go run ./cmd/gui
```

---

## GUI overview

The window is organised around three actors (see `testsheet.md` § 2):

- **Ground Control** — whitelist admin via `GC_ENDPOINT` + `GC_API_TOKEN`.
- **User** — create CIDs, approve/honor offers, manage epochs and funds.
- **Provider** — submit offers, withdraw, monitor open windows.
- **Demo** — side-by-side User and Provider workspaces for end-to-end testing.

For full CLI/GC flows and command reference, see [README.md](README.md) and [testsheet.md](testsheet.md).

---

## IPFS daemon

User workflows (load files to IPFS, create CIDs, check pins, etc.) need a local IPFS node. The GUI can start and stop one for you instead of running `ipfs daemon` in a separate terminal.

### Automatic start and stop

When the main window opens, the GUI checks whether the IPFS HTTP API is reachable at `http://127.0.0.1:5001`. If nothing is listening, it launches `ipfs daemon` as a child process and waits up to a few seconds for the API to come up.

When you close the GUI, it stops **only** the daemon it started. If you already had IPFS running before opening DCS (for example from a terminal), that external process is left running.

### Manual control (Settings)

Open **Settings** from the gear icon in the app bar. The **IPFS daemon** row shows:

| Indicator | Meaning |
|-----------|---------|
| Green dot + **Running** | The API at `127.0.0.1:5001` is up. |
| Red dot + **Stopped** | No local IPFS API detected. |

| Button | Action |
|--------|--------|
| **Turn on** | Start a managed `ipfs daemon` (same as auto-start). |
| **Turn off** | Stop the daemon the GUI started. |
| **Running** (disabled) | IPFS is up but was started outside DCS — stop it in your terminal if you need to. |

Use **Turn off** when you want to free the port or stop the node without quitting the whole app. Use **Turn on** if auto-start failed or you stopped it earlier.

### Requirements

- Install [Kubo (go-ipfs)](https://docs.ipfs.tech/install/) so `ipfs` is on `PATH`, or set **`IPFS_BIN`** in `.env` to the full path of the executable (for example `C:\Program Files\Kubo\ipfs.exe` on Windows).
- The GUI runs `ipfs daemon` with default Kubo settings (API on port **5001**). If another service already uses that port, start IPFS yourself or free the port before using **Turn on**.

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| Cross-compile / wrong OS binary | Build on the target OS (or matching CI runner); GUI needs CGO, unlike the CLI. |
| Build fails with CGO / gcc errors | Install the C toolchain on the **build** machine; set `CGO_ENABLED=1`. |
| Identity fields empty on launch | Run from the folder that contains `.env`, or set OS env vars. |
| GC whitelist actions fail | Set `GC_API_TOKEN` in `.env` (in addition to `GC_ENDPOINT`). |
| IPFS daemon won't start | Install IPFS and add it to `PATH`, or set `IPFS_BIN`. Check port 5001 is free. |
| **Turn off** unavailable | IPFS was started outside DCS — stop it in your terminal. |
| IPFS shows running but uploads fail | Wait a few seconds after **Turn on**, or restart from Settings. |
| Locale warnings on Linux/WSL | Usually harmless; the app normalises `LANG` on startup. |
