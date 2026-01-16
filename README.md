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

Windows (PowerShell):
```
copy .env.example .env
```
Linux (bash):
```
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
Windows (PowerShell):
```
cd C:\DCS
.\dcs.exe --help
.\dcs.exe iota_sc --help
```
Linux(bash):
```
cd ~/dcs
chmod +x ./dcs
./dcs --help
./dcs iota_sc --help
```
---
## Run from source:
### Clone the repository 
```
git clone https://github.com/teleconsys/DCS
cd DCS
go run ./cmd/cli --help
go run ./cmd/cli iota_sc --help
```
Build local binaries
Linux:
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dcs ./cmd/cli
```
Windows:
```
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dcs.exe ./cmd/cli
```

For full end-to-end flows and command testing, see: **`testsheet.md`**.