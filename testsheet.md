## 1. Introduction

This document provides a comprehensive checklist of CLI commands and their options for testing DCS (Distributed Content Security). It serves as a reference for the full command set and includes practical test scenarios that demonstrate typical usage.

## 2. Actors

The DCS system involves three main actors, each with distinct roles and responsibilities:

### 2.1. Ground Control (GC)

The administrative actor and on-chain owner of the deployed smart contract. GC is responsible for system-wide management and can:

- Add or remove users and providers from the whitelist. These operations require the GC signature, so the CLI forwards the request to the GC API deployed on a self-hosted server.

### 2.2. User

An actor who wants to store and manage content using DCS. Users can:

- Create an account to interact with the system
- View the COIN_IDs associated with their account
- Create CID objects for their content
- Approve provider offers for storing their CIDs
- Honor approved offers
- Transition epochs for CID objects they own
- Add funds to CID objects they own
- Remove CID objects from the system (if they own them)
- Check whether a CID is listed in the system

### 2.3. Provider

A storage service provider who offers storage capacity for CIDs in exchange for payments in IOTA tokens. Providers can:

- Create an account to interact with the system
- Submit storage offers for CIDs
- Withdraw payments after offers are honored and approved
- Monitor open offer windows


## 3. CLI command lists

This section provides a comprehensive list of all available CLI commands with their options and descriptions. The commands are shown as they would be executed by a user running the downloaded executable from a GitHub Release.

### 3.1. Ping

```powershell
./dcs iota_sc ping [--timeout <duration>]
```

**Description:**

- Quick connectivity check against the IOTA network
- Verifies that the RPC endpoint is accessible

**Options:**

- `--timeout <duration>`: RPC timeout (default: 5s)

### 3.2. Account

#### 3.2.1. Create New Account

```powershell
./dcs iota_sc account new --alias <name> [--no_faucet] [--faucet-amount <n>]
```

**Description:**

- Generate an ed25519 keypair and address
- Optionally fund via faucet from environment variables
- Saves account information to `./accounts/<alias>.json`

**Options:**

- `--alias <name>`: Account alias (required)
- `--no_faucet`: Do not call the faucet even if FAUCET_URL is set
- `--faucet-amount <n>`: Optional amount to request from faucet

**Environment Variables:**

- `FAUCET_URL`: Faucet endpoint URL

#### 3.2.2. List Account Coins (COIN_IDs)

```powershell
./dcs iota_sc account coins [--alias <name>] [--address <0x...>] [--timeout <duration>]
```

**Description:**

- Lists all coin object IDs owned by an address (including balances and coin types)

- Prints a recommended gas coin ID (largest IOTA coin)

**Options:**

- `--alias <name>`: Reads the address from ./accounts/<alias>.json

- `--address <0x...>`: Address to query directly (overrides --alias)

- `--timeout <duration>`: RPC timeout (default: 25s)

**Requirements:**

If using `--alias`, the file `./accounts/<alias>.json` must exist and contain at least the address field.

If you do not have an account file, use `--address` instead.

**Account File Format:**

Minimum required fields for `coins` command:
```
{
  "address": "0x..."
}
```

Typical fields (created automatically by `account new`):
```
{
  "alias": "test",
  "address": "0x...",
  "public_key": "...",
  "private_key": "...",
  "gas_coin_id": "0x..."
}
```

### 3.3. Whitelist

#### 3.3.1. Check if Address is in Whitelist

```powershell
./dcs iota_sc whitelist has [ADDRESS] [--member <address>] [--print-addr] [--id <whitelist-id>]
```

**Description:**

- Return true if the given ADDRESS is in the whitelist (RPC)
- **Note:** At least one of `ADDRESS` (positional argument) or `--member` flag must be provided

**Options:**

- `[ADDRESS]`: Address to check (0x...) - either this or `--member` must be specified
- `--member, -m <address>`: Address/ID to check (0x...) - either this or `ADDRESS` must be specified
- `--print-addr`: Print the address if present (instead of true/false)
- `--id <whitelist-id>`: Whitelist object ID (0x...) to use for this command. Takes priority over `DCS_WHITELIST_ID` env var

**Environment Variables:**

- `DCS_WHITELIST_ID`: Whitelist object ID

#### 3.3.2. Add Address to Whitelist

```powershell
./dcs iota_sc whitelist add [ADDRESS] [--member <address>] [--gc-endpoint <url>] [--gc-token <token>] [--id <whitelist-id>]
```

**Description:**

- Adds a user/provider address to the whitelist (requires GC signature)
- The CLI does not sign this transaction locally: it forwards the request to the GC API, which signs and submits the transaction on-chain
- **Note:** At least one of `ADDRESS` (positional argument) or `--member` flag must be provided

**Options:**

- `[ADDRESS]`: Address to add (0x...) - either this or `--member` must be specified
- `--member, -m <address>`: Address/ID to add (0x...) - either this or `ADDRESS` must be specified
- `--gc-endpoint <url>`: GroundControl API base URL (overrides GC_ENDPOINT env var)
- `--gc-token <token>`: GroundControl API token (overrides GC_API_TOKEN env var)
- `--id <whitelist-id>`: Whitelist object ID (0x...) to use for this command. Takes priority over `DCS_WHITELIST_ID` env var

**Environment Variables:**

- `GC_ENDPOINT`: Default GC API base URL (e.g. http://<VPS_IP>:8080)
- `GC_API_TOKEN`: Token used to authenticate requests to the GC API
- `DCS_WHITELIST_ID`: Default whitelist object ID (0x...) if --id is not provided

#### 3.3.3. Remove Address from Whitelist

```powershell
./dcs iota_sc whitelist remove [ADDRESS] [--member <address>] [--gc-endpoint <url>] [--gc-token <token>] [--id <whitelist-id>]
```
**Description:**

- Removes a user/provider address from the whitelist (requires GC signature)
- The CLI does not sign this transaction locally: it forwards the request to the GC API, which signs and submits the transaction on-chain
- **Note:** At least one of `ADDRESS` (positional argument) or `--member` flag must be provided

**Options:**

- `[ADDRESS]`: Address to add (0x...) - either this or `--member` must be specified
- `--member, -m <address>`: Address/ID to add (0x...) - either this or `ADDRESS` must be specified
- `--gc-endpoint <url>`: GroundControl API base URL (overrides GC_ENDPOINT env var)
- `--gc-token <token>`: GroundControl API token (overrides GC_API_TOKEN env var)
- `--id <whitelist-id>`: Whitelist object ID (0x...) to use for this command. Takes priority over `DCS_WHITELIST_ID` env var

**Environment Variables:**

- `GC_ENDPOINT`: Default GC API base URL (e.g. http://<VPS_IP>:8080)
- `GC_API_TOKEN`: Token used to authenticate requests to the GC API
- `DCS_WHITELIST_ID`: Default whitelist object ID (0x...) if --id is not provided

### 3.4. CID Management

#### 3.4.1. Create CID

```powershell
./dcs iota_sc cid create --type <path|cid> [CID] --epoch-start <timestamp> --epoch-end <timestamp> [--amount <amount>] [--signer-address <address>] [--signer-private-key <key>] [--signer-gas-id <coin-id>]
```

**Description:**

- Create a new CID object in the smart contract
- Input can be a CID string or a local file path (uploaded to IPFS)
- Creates a new coin of `--amount` and associates it with the new CID object
- If `--epoch-start` 0 and `--epoch-end` 0 are used, the CLI auto-generates a default epoch window

**Options:**

- `--type <path|cid>`: Type of input - 'path' to upload file, 'cid' for existing CID (required)
- `[CID]`: CID string (if --type is 'cid') or file path (if --type is 'path')
- `--epoch-start <timestamp>`: Next epoch start timestamp (required, 0 for default duration)
- `--epoch-end <timestamp>`: Next epoch end timestamp (required, 0 for default duration)
- `--amount <amount>`: Amount for the new coin associated to the CID object (IOTA nanos, default: 100000)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `USER_PRIVATE_KEY`); if omitted you will be prompted to insert it (supports bech32/base64/hex formats)
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `USER_ADDRESS`)
- `--signer-gas-id <coin-id>`: signer’s existing gas coin used to pay fees and to split out the --amount coin (overrides `ACTIVE_GAS_COIN_ID` / `USER_GAS_COIN_ID`)

#### 3.4.2. Remove CID

```powershell
./dcs iota_sc cid remove [objectId] [--signer-address <address>] [--signer-private-key <key>] [--signer-gas-id <coin-id>]
```

**Description:**

- Remove a CID listed in the smart contract
- The signer must be the owner of the CID object to remove it

**Options:**

- `[objectId]`: CID object ID (0x...) (required)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `USER_PRIVATE_KEY`); if omitted you will be prompted to insert it
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `USER_ADDRESS`)
- `--signer-gas-id <coin-id>`: Gas coin object ID used to pay for the transaction (overrides `ACTIVE_GAS_COIN_ID` / `USER_GAS_COIN_ID`)

#### 3.4.3. Check if CID is in List

```powershell
./dcs iota_sc cid is-in-list [objectId]
```

**Description:**

- Check if a CID ID is listed in the smart contract

**Options:**

- `[objectId]`: CID object ID (0x...) (required)

#### 3.4.4. Transition to Next Epoch

```powershell
./dcs iota_sc cid next-epoch [objectId] [--signer-private-key <key>] [--signer-address <address>] [--signer-gas-id <coin-id>]
```

**Description:**

- Transition to the next epoch for a CID object
- **Notes:** 
  - An "epoch" is the CID’s time window used by the smart contract to enforce lifecycle rules over time. There are 3 main epoch windows next, current and previuos. Transitions moves next->current->previus and creates a new next based on the lenght of the epochs defined in CID creation. 
  - The transition uses the on-chain clock (Clock object 0x6) to evaluate timing and update the CID’s epoch-related state
  - The CLI checks the CID’s stored next_epoch_start timestamp: if the next epoch has not started yet, it prints a warning and exits without submitting a transaction

**Options:**

- `[objectId]`: CID object ID (0x...) (required)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `USER_PRIVATE_KEY`); if omitted you will be prompted to insert it
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `USER_ADDRESS`)
- `--signer-gas-id <coin-id>`: Gas coin object ID used to pay for the transaction (overrides `ACTIVE_GAS_COIN_ID` / `USER_GAS_COIN_ID`)

#### 3.4.5. Add Funds to CID

```powershell
./dcs iota_sc cid add-funds [objectId] --amount <amount> [--signer-address <address>] [--signer-private-key <key>] [--signer-gas-id <coin-id>]
```

**Description:**

- Deposit IOTA coins into an owned CID object

**Options:**

- `[objectId]`: CID object ID (0x...) (required)
- `--amount <amount>`: Amount of money to transfer to the CID object. A new coin object id will be created and deposit into the CID object (0x...). This gas coin object will be entirely consumed and deleted after the transaction is executed. (required)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `USER_PRIVATE_KEY`); if omitted you will be prompted to insert it
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `USER_ADDRESS`)
- `--signer-gas-id <coin-id>`: Gas coin object ID used to pay for the transaction (overrides `ACTIVE_GAS_COIN_ID` / `USER_GAS_COIN_ID`)

### 3.5. Offer Management

#### 3.5.1. Submit Offer

```powershell
./dcs iota_sc submit_offer --cid <cid> --amount <amount> [--signer-address <address>] [--signer-private-key <key>] [--signer-gas-id <coin-id>] [--debug]
```

**Description:**

- Submit an offer for the next epoch (provider wallet)
- **Notes:**: 
  - Every epoch of a CID object has its window open for offering which is 10 min after next_epoch_start. Calling the command out of the window will result in a failure with a detailed error message (telling also when next window will open for that CID)
  - Non approved offers after the window closes will be automatically be considered discarded

**Options:**

- `--cid <cid>`: CID object id (0x...) (required)
- `--amount <amount>`: Offer amount (IOTA nanos) (required)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `PROVIDER_PRIVATE_KEY`); if omitted you will be prompted to insert it
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `PROVIDER_ADDRESS`)
- `--signer-gas-id <coin-id>`: Gas coin object ID used to pay for the transaction (overrides `ACTIVE_GAS_COIN_ID` / `PROVIDER_GAS_COIN_ID`)
- `--debug`: Verbose debug (preflight + postflight)

#### 3.5.2. Approve Offer

```powershell
./dcs iota_sc approve_offer --cid <cid> --idx <index> [--signer-address <address>] [--signer-private-key <key>] [--signer-gas-id <coin-id>] [--debug]
```

**Description:**

- Approve a provider offer in next_epoch_offers (CID owner)

**Options:**

- `--cid <cid>`: CID object id (0x...) (required)
- `--idx <index>`: Offer index in next_epoch_offers to approve (0-based)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `USER_PRIVATE_KEY`); if omitted you will be prompted to insert it
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `USER_ADDRESS`)
- `--signer-gas-id <coin-id>`: Gas coin object ID used to pay for the transaction (overrides `ACTIVE_GAS_COIN_ID` / `USER_GAS_COIN_ID`)
- `--debug`: Verbose debug

#### 3.5.3. Honor Offer

```powershell
./dcs iota_sc honor_offer --cid <cid> --idx <index> [--signer-address <address>] [--signer-private-key <key>] [--signer-gas-id <coin-id>] [--debug]
```

**Description:**

- Honor an accepted offer in the current epoch (CID owner)
- Can be done only during current epoch, otherwise an error will be printed 

**Options:**

- `--cid <cid>`: CID object id (0x...) (required)
- `--idx <index>`: Offer index in next_epoch_offers to approve (0-based)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `USER_PRIVATE_KEY`); if omitted you will be prompted to insert it
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `USER_ADDRESS`)
- `--signer-gas-id <coin-id>`: Gas coin object ID used to pay for the transaction (overrides `ACTIVE_GAS_COIN_ID` / `USER_GAS_COIN_ID`)
- `--debug`: Verbose debug

#### 3.5.4. List Open Offers

```powershell
go run main.go iota_sc list-open-offers [--graphql-endpoint <url>] [--cidlist-id <id>]
```

**Description:**

- Print CIDs whose offer window is open now
- Shows active offers and the time remaining to approve them

**Options:**

- `--graphql-endpoint <url>`: GraphQL endpoint (overrides .env)
- `--cidlist-id <id>`: CID list object ID (overrides .env)

**Environment Variables:**

- `IOTA_GRAPHQL_ENDPOINT`: GraphQL endpoint (default: `https://graphql.testnet.iota.cafe`)
- `DCS_CIDLIST_ID`: CID list object ID

#### 3.5.5. Withdraw

```powershell
./dcs iota_sc withdraw --cid <cid> --idx <index> [--signer-address <address>] [--signer-private-key <key>] [--signer-gas-id <coin-id>] [--debug]
```

**Description:**

- Withdraw IOTA from a fulfilled offer
- Offer must be honored
- This command can be only called after the current epoch ended (So in current version the offer must be in the `previous_epoch` offers list)

**Options:**

- `--cid <cid>`: CID object id (0x...) (required)
- `--idx <index>`: Payment index to withdraw (0-based) (required)
- `--signer-private-key <key>`: Private key for signing (overrides `ACTIVE_PRIVATE_KEY` / `PROVIDER_PRIVATE_KEY`); if omitted you will be prompted to insert it
- `--signer-address <address>`: Address of the signer (overrides `ACTIVE_ADDRESS` / `PROVIDER_ADDRESS`)
- `--signer-gas-id <coin-id>`: Gas coin object ID used to pay for the transaction (overrides `ACTIVE_GAS_COIN_ID` / `PROVIDER_GAS_COIN_ID`)
- `--debug`: Verbose debug

## 4. Utils from Iota CLI

Utility commands for inspecting and monitoring the system state using the IOTA client CLI. 
**NOTE:**
All the following commands will not work if the [`iota-cli`](https://docs.iota.org/developer/references/cli) is not installed. These commands fall ouside of the project scope, but are indeed useful for additional testing options. 

### 4.1. Inspect Object

```bash
iota client object <object-id> [--json]
```

**Description:**

- Shows all information about an object on the IOTA network
- With `--json` flag, outputs data in JSON format for easier parsing
- Useful for inspecting CID objects to see:
  - `current_epoch_offers` or `prev_epochs...` arrays
  - State of the flags: `approved`, `honored`, `paid`
  - Epoch timing information
  - Other object metadata

**Example:**

```bash
iota client object 0x87b8e522174e753ab17f67f353a339c9df16f893fa5d679b604c1f197107cbd0 --json
```

### 4.2. Gas

```bash
iota client gas [<address>]
```

**Description:**

- Check gas balance for an address or the default signer
- Shows available gas coins that can be used for transactions
- Useful for verifying that accounts have sufficient funds for operations

**Example:**

```bash
iota client gas
iota client gas 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b
```

### 4.3. Faucet

```bash
iota client faucet --address <address>
```

**Description:**

- Request test tokens from the IOTA testnet faucet
- Useful for obtaining initial funds for new accounts
- Funds an address with test IOTA tokens for development and testing purposes

**Options:**

- `--address <address>`: Address to fund (0x...)

**Example:**

```bash
iota client faucet --address 0x731f57d2f3c5b102e5a4d182c8d4b6c06f0ba3aaf5534e1913e99631163d3edd
```

### 4.4. Import Key

```bash
iota keytool import <private-key> ed25519 --alias <name>
```

**Description:**

- Import a private key into the IOTA client keychain
- Creates a named alias for the imported key
- Allows using the key with the `--alias` flag in subsequent commands

**Options:**

- `<private-key>`: Private key to import (iotaprivkey1...)
- `ed25519`: Key type (ed25519)
- `--alias <name>`: Alias name for the imported key

**Example:**

```bash
iota keytool import iotaprivkey1qp5n5ermut5gvfnvcdp5yhdk50jkdurqydyu7pz3e5qysdvpe6xtqka84au ed25519 --alias dcs_cid_owner
```

### 4.5. Switch Address

```bash
iota client switch --address <address>
```

**Description:**

- Switch the default signer address for the IOTA client
- Sets the active address that will be used for transactions when no address is explicitly specified
- Useful for managing multiple accounts

**Options:**

- `--address <address>`: Address to set as default (0x...)

**Example:**

```bash
iota client switch --address 0x731f57d2f3c5b102e5a4d182c8d4b6c06f0ba3aaf5534e1913e99631163d3edd
```

### 4.6. List Addresses

```bash
iota client addresses
```

**Description:**

- List all addresses associated with the current IOTA client configuration
- Shows addresses that have been imported or created
- Useful for verifying available accounts

**Example:**

```bash
iota client addresses
```

### 4.7. Create Gas Coin Object

```bash
iota client pay-iota --input-coins <coin-id> --amounts <amount> --recipients <address>
```

**Description:**

- Transfer IOTA tokens and create a new gas coin object
- Splits an existing coin into a new gas coin object with a new ID
- The new gas coin object ID is automatically generated (cannot be specified)
- Useful for creating gas coins for transactions

**Options:**

- `--input-coins <coin-id>`: Input coin object ID to spend from (0x...)
- `--amounts <amount>`: Amount to send (in nanos)
- `--recipients <address>`: Recipient address (0x...)

**Example:**

```bash
iota client pay-iota --input-coins 0x55ed45ebd47a7c315856871b190e35b1457852862fa42aeb002faf37e1bea90a --amounts 16000 --recipients 0xea8180205d4c42f165083d2a42da34b329ac866f8c0eccb9b458bb6a41009296
```

**Notes:**

- Creates a new gas coin object with a new ID (the ID cannot be specified)

## 5. Main Test Scenario

This section walks through an end-to-end test of the CID lifecycle using the CLI. The flow starts from a clean setup (accounts, gas coin discovery, and whitelist enrollment) and then exercises the full on-chain offer process: a user creates a CID, a provider submits an offer, the user approves and honors it across epoch boundaries, and finally the provider withdraws the payment.

All commands are written with placeholders (e.g. `<CID_OBJECT_ID>`, `<USER_ADDRESS>`) so you can reuse the same scenario with any accounts and any CID.

**Timing model used in this scenario:**

- Offers must be submitted during the “offer window” at the beginning of an epoch (wait for the window to close before approving).
- After an offer is approved, it is honored in the following epoch; epoch transitions become effective only after the chain time has passed the epoch boundary.

### 5.0. Setup (accounts + whitelist)

This section prepares the environment for the full CID lifecycle scenario:
- Create and fund accounts (user + provider)
- Discover COIN_IDs for gas
- Add both addresses to the whitelist via the GC API
- Switch the active signer between user and provider during the test

### 5.0.1. Create User Account (user)

```powershell
./dcs iota_sc account new --alias test 
```
**Note:**
- Creates `./accounts/test.json` in the current working directory

### 5.0.2. List User COIN_IDs (user)
```powershell
./dcs iota_sc account coins --alias test
```
**Note:**
- Prints a recommended gas coin ID and a copy/paste line like: `export USER_GAS_COIN_ID=0x...`
- Use that COIN_ID as the user gas coin (`USER_GAS_COIN_ID` in .env)

### 5.0.3. Create Provider Account (provider) (optional)
```powershell
./dcs iota_sc account new --alias provider 
```
**Notes:**
- Optional: only needed if you want to test the full provider flow using a fresh provider account
- Creates `./accounts/provider.json` in the current working directory

### 5.0.4. List Provider COIN_IDs (provider) (optional)
```powershell
./dcs iota_sc account coins --alias provider
```
**Note:**
- Use the recommended COIN_ID as PROVIDER_GAS_COIN_ID in .env

### 5.0.5. Add User/Provider to Whitelist (GC)
```powershell
./dcs iota_sc whitelist add <USER_ADDRESS> --gc-endpoint <GC_ENDPOINT> --gc-token <GC_API_TOKEN>
./dcs iota_sc whitelist add <PROVIDER_ADDRESS> --gc-endpoint <GC_ENDPOINT> --gc-token <GC_API_TOKEN>
```
**Note:**
- If you already set `GC_ENDPOINT` and/or `GC_API_TOKEN`(this is not necessary) in .env, you can omit the flags: `--gc-endpoint` and `--gc-token`

### 5.0.6. Switch Active Signer (user ↔ provider)

Use this when a command needs a signature and you want to force which identity signs.
```powershell
# Switch to USER signer
$env:ACTIVE_ADDRESS="<USER_ADDRESS>"
$env:ACTIVE_PRIVATE_KEY="<USER_PRIVATE_KEY>"
$env:ACTIVE_GAS_COIN_ID="<USER_GAS_COIN_ID>"

# Switch to PROVIDER signer
$env:ACTIVE_ADDRESS="<PROVIDER_ADDRESS>"
$env:ACTIVE_PRIVATE_KEY="<PROVIDER_PRIVATE_KEY>"
$env:ACTIVE_GAS_COIN_ID="<PROVIDER_GAS_COIN_ID>"
```

### 5.1. Create CID (user)

```powershell
./dcs iota_sc cid create <YOUR_CID_STRING> --epoch-start 0 --epoch-end 0 --type cid
```
**Note:**
- This command will print the CID object ID. Keep it handy as it will be used later on

### 5.2. Submit Offer (provider)

```bash
./dcs iota_sc submit_offer --cid <CID_OBJECT_ID> --amount 100000
```

**Notes:**

- Must be executed within the time limit (before the timer from the following command is expired)
- After the command, the offer is in the **next** epoch, with `approved: false`, `honored: false`, `paid: false`

### 5.3. List Open Offers

```powershell
./dcs iota_sc list-open-offers
```

**Notes:**

- Shows active offers and the time after which they can be approved

### 5.4. Approve Offer (user)

```powershell
./dcs iota_sc approve_offer --cid <CID_OBJECT_ID> --idx 0
```

**Notes:**

- Execute immediately after the time limit for submitting offers has expired
- The command approves the offer in slot 0 of the list, so the first received.
- After the command, the offer is in the **next** epoch, with `approved: true`, `honored: false`, `paid: false`

### 5.5. Transition to Next Epoch (user)

```powershell
./dcs iota_sc cid next-epoch <CID_OBJECT_ID>
```

**Notes:**

- Execute immediately after the previous step
- Only effective after the actual end of the current epoch
- After the command, the offer is in the **current** epoch, with `approved: true`, `honored: false`, `paid: false`

### 5.6. Honor Offer (user)

```powershell
./dcs iota_sc honor_offer --cid <CID_OBJECT_ID> --idx 0
```

**Notes:**

- Must be executed before the end of the current epoch
- The command only honors the first approved offer
- After the command, the offer is in the **current** epoch, with `approved: true`, `honored: true`, `paid: false`

### 5.7. Transition to Next Epoch (Again) (user)

```powershell
./dcs iota_sc cid next-epoch <CID_OBJECT_ID>
```

**Notes:**

- Only effective after the actual end of the current epoch
- After the command, the offer is in the **previous** epoch, with `approved: true`, `honored: true`, `paid: false`

### 5.8. Withdraw (provider)

```powershell
./dcs iota_sc withdraw --cid <CID_OBJECT_ID> --idx 0
```

**Notes:**

- Execute after the ex-current epoch (now previous epoch) has ended and before transitioning to the next one
- This command only withdraws from the first honored offer on that CID. If multiple honored offers were active for that CID, withdraw must be call multiple times.
- After the command, the offer is in the **previous** epoch, with `approved: true`, `honored: true`, `paid: true`



