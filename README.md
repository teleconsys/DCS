# DCS
Distributed Content Security: IOTA and IPFS for a secure and distributed storage solution 

# DCS (CLI + GC API) — IOTA Rebased

DCS is a Go-based CLI that interacts with:
- a IOTA Rebased Move smart contract deployed on testnet [smart contract](https://github.com/SDV-Consulting/dcs)

- IPFS (InterPlanetary File System)
---
## Quick start: download the executable 
### 1) Download from GitHub Releases (recommended)
Go to this repository’s **Releases** page and download the artifact for your OS:

- **Windows**: `dcs-windows-amd64.zip`
- **Linux**: `dcs-linux-amd64.tar.gz`

Unzip/extract into a folder of your choice, for example:
- Windows: `C:\DCS\`
- Linux: `~/dcs/`

Your folder should look like:
```
dcs.exe (Windows)  OR  dcs (Linux)
.env.example
```
`.env.example`
```
# IOTA Rebased testnet RPC
REBASE_RPC=https://api.testnet.iota.cafe:443
FAUCET_URL=https://faucet.testnet.iota.cafe
IOTA_GRAPHQL_ENDPOINT=https://graphql.testnet.iota.cafe

# GC API Endopoint
GC_ENDPOINT=http://87.106.12.48:8080

# Public Network info
DCS_PACKAGE_ID=0x6f1171331dd83efac6cecba51a5943d50553c2728554c70cd7d4d37dfa0eccde
DCS_WHITELIST_ID=0x544d411977ea6255686388050b40f5652207ee12e0b3cb6e7d8ce1bea98f0bc7
DCS_CIDLIST_ID=0x17ee700860a8b106b51430839996125a074a609b29a834b24d60404d32753e39
DCS_CLOCK_ID=0x6
WALLET_GAS_BUDGET=10000000

# Account User
USER_ADDRESS=0xb7d72d119424319aae4dd1431a382ac6e870ffb4aa2e44b1572aa59f5d084d6f
USER_PRIVATE_KEY=iotaprivkey1qp8j2zn56rvdd332yhy39zvmmdn9reykhmvsfg8mckvh2f7jqghzylzen24
USER_GAS_COIN_ID=0x04a0a4e93f42e3eb825fae82ed6e10410f22d706fc73c7e840b2eae42524bf18

# Account Provider
PROVIDER_PRIVATE_KEY=iotaprivkey1qzkpuj2kpee39r0rezlk63c3s5v3y0pf8wj59ew9dfhhqgjnsfne64dqhy7
PROVIDER_ADDRESS=0xef58bc2741cb4924aea74f860815f0a77a83db52b4921507ba78a589fab87f5a
PROVIDER_GAS_COIN_ID=0x4c7414b170e87636dd84cb5052e0a927ab3114d4cd065278a596fa561c3e3bf0
```
USER_* and PROVIDER_* are example/test identities used for local signing on testnet that can be used as reference.
Replace them with your own identities for real testing.

### 2) Configure .env

Create your own `.env` by copying `.env.example`:

Windows:
```powershell
copy .env.example .env
```
Linux:
```bash
cp .env.example .env
```
Then edit .env and set the values you need.

Value formats:
Addresses and object IDs use the Sui-style hex format:

    USER_ADDRESS=0x...

    USER_GAS_COIN_ID=0x... (this is a coin object ID, not a balance)

Private keys:

    USER_PRIVATE_KEY=iotaprivkey1... (recommended; bech32 format)

    The CLI also accepts (depending on the command) either:

        base64 keystore format (33 bytes: [0x00 | 32-byte seed] encoded as base64)

### 3) Run
Windows:
```powershell
cd C:\DCS
.\dcs.exe --help
.\dcs.exe iota_sc --help
```
Linux:
```bash
cd ~/dcs
chmod +x ./dcs
./dcs --help
./dcs iota_sc --help
```
---
## Run from source:
### Clone the repository 
```bash
git clone https://github.com/teleconsys/DCS
cd DCS
go run ./cmd/cli --help
go run ./cmd/cli iota_sc --help
```
Build local binaries
Linux:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dcs ./cmd/cli
```
Windows:
```powershell
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dcs.exe ./cmd/cli
```

For full end-to-end flows and command testing, see: **`testsheet.md`**.

---

## Desktop GUI (`dcs-gui`)

Alongside the CLI, the repository ships a Fyne-based desktop front-end at
`cmd/gui`. It exposes every CLI command through forms and runs them
against the same `internal/*` packages, so the CLI and the GUI are
guaranteed to share behaviour.

### Actor model

The GUI is organized around the three actors described in `testsheet.md`
§ 2:

- **Ground Control** — admin; uses `GC_ENDPOINT` + `GC_API_TOKEN` to add
  or remove whitelist members via the GC HTTP API (no on-chain signing).
- **User** — content owner; signs with `USER_PRIVATE_KEY` /
  `ACTIVE_PRIVATE_KEY`. Creates CIDs, approves and honors offers,
  transitions epochs, adds funds, removes CIDs.
- **Provider** — storage provider; signs with `PROVIDER_PRIVATE_KEY` /
  `ACTIVE_PRIVATE_KEY`. Submits offers, withdraws after honor, monitors
  open windows.

A top-level switcher toggles between **Ground Control**, **User**,
**Provider**, and a **Demo: User ⟷ Provider** mode that hosts a full
User workspace on the left and a full Provider workspace on the right
for end-to-end demos. Each side runs its own actions independently;
every service call snapshots its actor profile at invocation time, so
the two sides never trample each other.

Accounts created from the GUI (`Account: new` view) are tagged with a
`role` field (`user` or `provider`) inside `accounts/<alias>.json`. The
per-actor alias picker filters to matching files first and offers
untagged accounts (created by the CLI) as a fallback. Roles are
advisory — not enforced — so the CLI and the GUI remain interoperable.

### Build prerequisites

Fyne requires `CGO_ENABLED=1` and a working C toolchain:

- **Windows**: install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or
  mingw-w64 via `winget`:
  ```powershell
  winget install -e --id BrechtSanders.WinLibs.POSIX.UCRT
  ```
  Then make sure `gcc.exe` is on `PATH`.
- **Linux** (Debian/Ubuntu):
  ```bash
  sudo apt install gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev
  ```
- **macOS**: the Xcode Command Line Tools are sufficient (`xcode-select --install`).

### First build

The first build also needs to fetch the Fyne module and refresh
`go.sum`:

```bash
go mod tidy
```

Then build and run:

Linux:
```bash
CGO_ENABLED=1 go build -o dcs-gui ./cmd/gui
./dcs-gui
```

Windows (PowerShell):
```powershell
$env:CGO_ENABLED = "1"
go build -o dcs-gui.exe ./cmd/gui
.\dcs-gui.exe
```

Run the binary from a folder that contains your `.env` (same expectation
as the CLI). On startup the GUI reads the environment variables to
pre-fill the per-actor identity panels; you can override any field
inline.

### Where to start

- Use the **User** workspace if you want to upload content and manage
  CIDs (load a file → IPFS, create CID, approve/honor offers, …).
- Use the **Provider** workspace to monitor open offer windows, submit
  offers, and withdraw payments.
- Use the **Demo** mode to drive the full User⟷Provider flow in one
  window (handy for screenshots, recordings, or local end-to-end
  testing).