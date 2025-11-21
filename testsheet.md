# Testsheet DCS

## 1. Introduction

This document provides a comprehensive checklist of CLI commands and their available options for testing the DCS (DeCentralized Storage). The document serves as a general reference for all CLI commands, with detailed test scenarios demonstrating their usage.

## 2. Actors

The DCS system involves three main actors, each with distinct roles and responsibilities:

### Ground Control (GC)

Admin actor responsible for system-wide management. GC can:

- Add or remove providers from the whitelist

**Environment Variables:**

- `GC_PRIVATE_KEY`: Ground Control private key (iotaprivkey1...)
- `GC_ADDRESS`: Ground Control address (0x...)
- `GC_WALLET_GAS_ID`: Gas coin object ID for GC operations

### User

Actor who wants to store their content in a decentralized manner. Users can:

- Create CID objects for their content
- Approve provider offers for storage
- Honor approved offers by providing the content
- Transition epochs for their CID objects
- Add funds to CID objects
- Remove CID objects from the system
- Check if a CID is listed in the system

**Environment Variables:**

- `USER_ADDRESS`: User address (0x...)
- `USER_PRIVATE_KEY`: User private key (iotaprivkey1...)
- `USER_GAS_COIN_ID`: Gas coin object ID for user operations

### Provider

Storage service provider actor who offers storage capacity for CIDs. Providers can:

- Submit storage offers for CIDs
- Withdraw payments after offers are honored and approved
- Monitor open offer windows

**Environment Variables:**

- `PROVIDER_ADDRESS`: Provider address (0x...)
- `PROVIDER_PRIVATE_KEY`: Provider private key (iotaprivkey1...)
- `PROVIDER_GAS_COIN_ID`: Gas coin object ID for provider operations

## 3. CLI command lists

This section provides a comprehensive list of all available CLI commands with their options and descriptions.

### 3.1. Ping

```bash
go run main.go iota_sc ping [--timeout <duration>]
```

**Description:**

- Quick connectivity check against the IOTA network
- Verifies that the RPC endpoint is accessible

**Options:**

- `--timeout <duration>`: RPC timeout (default: 5s)

### 3.2. Account

#### 3.2.1. Create New Account

```bash
go run main.go iota_sc account new --alias <name> [--no_faucet] [--faucet-amount <n>]
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

### 3.3. Whitelist

#### 3.3.1. Check if Address is in Whitelist

```bash
go run main.go iota_sc whitelist has [ADDRESS] [--member <address>] [--print-addr] [--id <whitelist-id>]
```

**Description:**

- Return true if ADDRESS/ID is in the whitelist (RPC)
- **Note:** At least one of `ADDRESS` (positional argument) or `--member` flag must be provided

**Options:**

- `[ADDRESS]`: Address to check (0x...) - either this or `--member` must be specified
- `--member, -m <address>`: Address/ID to check (0x...) - either this or `ADDRESS` must be specified
- `--print-addr`: Print the address if present (instead of true/false)
- `--id <whitelist-id>`: Whitelist object ID (0x...) to use for this command. Takes priority over `DCS_WHITELIST_ID` env var

**Environment Variables:**

- `DCS_WHITELIST_ID`: Whitelist object ID

#### 3.3.2. Add Address to Whitelist

```bash
go run main.go iota_sc whitelist add [ADDRESS] [--member <address>] [--package-id <id>] [--gas <coin-id>] [--gas-budget <amount>] [--id <whitelist-id>]
```

**Description:**

- Add ADDRESS/ID to the whitelist (requires signer)
- **Note:** At least one of `ADDRESS` (positional argument) or `--member` flag must be provided

**Options:**

- `[ADDRESS]`: Address to add (0x...) - either this or `--member` must be specified
- `--member, -m <address>`: Address/ID to add (0x...) - either this or `ADDRESS` must be specified
- `--package-id <id>`: DCS package ID (0x...)
- `--gas <coin-id>`: Gas coin object ID (0x...)
- `--gas-budget <amount>`: Gas budget (nanos)
- `--id <whitelist-id>`: Whitelist object ID (0x...) to use for this command. Takes priority over `DCS_WHITELIST_ID` env var

#### 3.3.3. Remove Address from Whitelist

```bash
go run main.go iota_sc whitelist remove [ADDRESS] [--member <address>] [--package-id <id>] [--gas <coin-id>] [--gas-budget <amount>] [--id <whitelist-id>]
```

**Description:**

- Remove ADDRESS/ID from the whitelist (requires signer)
- **Note:** At least one of `ADDRESS` (positional argument) or `--member` flag must be provided

**Options:**

- `[ADDRESS]`: Address to remove (0x...) - either this or `--member` must be specified
- `--member, -m <address>`: Address/ID to remove (0x...) - either this or `ADDRESS` must be specified
- `--package-id <id>`: DCS package ID (0x...)
- `--gas <coin-id>`: Gas coin object ID (0x...)
- `--gas-budget <amount>`: Gas budget (nanos)
- `--id <whitelist-id>`: Whitelist object ID (0x...) to use for this command. Takes priority over `DCS_WHITELIST_ID` env var

### 3.4. CID Management

#### 3.4.1. Create CID

```bash
go run main.go iota_sc cid create --type <path|cid> [CID] --epoch-start <timestamp> --epoch-end <timestamp> [--user-address <address>] [--user-private-key <key>] [--user-coin-id <coin-id>]
```

**Description:**

- Create a new CID object in the smart contract
- Can provide a CID directly or upload a file to IPFS

**Options:**

- `--type <path|cid>`: Type of input - 'path' to upload file, 'cid' for existing CID (required)
- `[CID]`: CID string (if --type is 'cid') or file path (if --type is 'path')
- `--epoch-start <timestamp>`: Next epoch start timestamp (required)
- `--epoch-end <timestamp>`: Next epoch end timestamp (required)
- `--user-private-key <key>`: Private key for signing(overrides USER_PRIVATE_KEY env var); if omitted you will be prompted to insert it <!-- TODO if omitted read from USER_PRIVATE_KEY -->
- `--user-address <address>`: Address of the user (overwrites USER_ADDRESS env var)
- `--user-coin-id <coin-id>`: Coin ID of the user (overwrites USER_GAS_COIN_ID env var)

#### 3.4.2. Remove CID

```bash
go run main.go iota_sc cid remove --cid-type <id|cid> [objectId|cid] [--user-address <address>] [--user-private-key <key>] [--user-coin-id <coin-id>]
```

**Description:**

- Remove a CID listed in the smart contract

**Options:**

- `--cid-type <id|cid>`: Type of cid - 'id' for object ID, 'cid' for CID string (required)
- `[objectId|cid]`: CID object ID or CID string
- `--user-private-key <key>`: Private key for signing (overrides USER_PRIVATE_KEY env var); if omitted you will be prompted to insert it
- `--user-address <address>`: Address of the user (overwrites USER_ADDRESS env var)
- `--user-coin-id <coin-id>`: Coin ID of the user (overwrites USER_GAS_COIN_ID env var)

#### 3.4.3. Check if CID is in List

```bash
go run main.go iota_sc cid is-in-list --cid-type <id|cid> [objectId|cid]
```

**Description:**

- Check if a CID ID is listed in the smart contract

**Options:**

- `--cid-type <id|cid>`: Type of cid - 'id' for object ID, 'cid' for CID string (required)
- `[objectId|cid]`: CID object ID or CID string

#### 3.4.4. Transition to Next Epoch

```bash
go run main.go iota_sc cid next-epoch --cid-type <id|cid> [objectId|cid] [--user-private-key <key>]
```

**Description:**

- Transition to the next epoch for a CID object

**Options:**

- `--cid-type <id|cid>`: Type of cid - 'id' for object ID, 'cid' for CID string (required)
- `[objectId|cid]`: CID object ID or CID string
- `--user-private-key <key>`: Private key for signing (overrides USER_PRIVATE_KEY env var); if omitted you will be prompted to insert it

#### 3.4.5. Add Funds to CID

```bash
go run main.go iota_sc cid add-funds --cid-type <id|cid> [objectId|cid] --coin-id <coin-id> [--user-address <address>] [--user-private-key <key>] [--user-gas-coin-id <coin-id>]
```

**Description:**

- Deposit IOTA coins into a CID

**Options:**

- `--cid-type <id|cid>`: Type of cid - 'id' for object ID, 'cid' for CID string (required)
- `[objectId|cid]`: CID object ID or CID string
- `--coin-id <coin-id>`: Coin object ID to deposit (0x...) (required)
- `--user-private-key <key>`: Private key for signing (overrides USER_PRIVATE_KEY env var); if omitted you will be prompted to insert it
- `--user-address <address>`: User signer address (0x...) overrides env
- `--user-gas-coin-id <coin-id>`: Gas coin object id (0x...) overrides env

### 3.5. Offer Management

#### 3.5.1. Submit Offer

```bash
go run main.go iota_sc submit_offer --cid <cid> --amount <amount> --cid-type <id|cid> [--signer-address <address>] [--signer-private-key <key>] [--gas-id <coin-id>] [--debug]
```

**Description:**

- Submit an offer for the next epoch (provider wallet)

**Options:**

- `--cid <cid>`: CID (object id 0x... or CID string) (required)
- `--amount <amount>`: Offer amount (IOTA nanos) (required)
- `--cid-type <id|cid>`: Interpret --cid as 'id' or 'cid' (default: 'id')
- `--signer-address <address>`: Signer address (0x...) overrides env
- `--signer-private-key <key>`: Signer private key (iotaprivkey1...); if omitted you will be prompted to insert it
- `--gas-id <coin-id>`: Gas coin object id (0x...) overrides env
- `--debug`: Verbose debug (preflight + postflight)

**Environment Variables:**

- `PROVIDER_ADDRESS`: Provider signer address
- `PROVIDER_GAS_COIN_ID`: Provider gas coin ID
- `DCS_WHITELIST_ID`: Whitelist object ID
- `DCS_CLOCK_ID`: Clock object ID (default: 0x6)
- `DCS_PACKAGE_ID`: DCS package ID
- `WALLET_GAS_BUDGET`: Gas budget (default: 10_000_000)

#### 3.5.2. Approve Offer

```bash
go run main.go iota_sc approve_offer --cid <cid> --idx <index> --cid-type <id|cid> [--signer-address <address>] [--signer-private-key <key>] [--gas-id <coin-id>] [--debug]
```

**Description:**

- Approve a provider offer in next_epoch_offers (CID owner)

**Options:**

- `--cid <cid>`: CID (object id 0x... or CID string) (required)
- `--idx <index>`: Offer index in next_epoch_offers to approve (0-based)
- `--cid-type <id|cid>`: Interpret --cid as 'id' or 'cid' (default: 'id')
- `--signer-address <address>`: Signer address (0x...) overrides env
- `--signer-private-key <key>`: Signer private key (iotaprivkey1...); if omitted you will be prompted to insert it
- `--gas-id <coin-id>`: Gas coin object id (0x...) overrides env
- `--debug`: Verbose debug

**Environment Variables:**

- `USER_ADDRESS`: User signer address
- `USER_GAS_COIN_ID`: User gas coin ID
- `DCS_CLOCK_ID`: Clock object ID (default: 0x6)
- `DCS_PACKAGE_ID`: DCS package ID
- `WALLET_GAS_BUDGET`: Gas budget (default: 10_000_000)

#### 3.5.3. Honor Offer

```bash
go run main.go iota_sc honor_offer --cid <cid> --idx <index> [--cid-type <id|cid>] [--signer-address <address>] [--signer-private-key <key>] [--gas-id <coin-id>] [--debug]
```

**Description:**

- Honor a confirmed offer in the current epoch (CID owner)

**Options:**

- `--cid <cid>`: CID (object id 0x... or CID string) (required)
- `--idx <index>`: Offer index in next_epoch_offers to approve (0-based)
- `--cid-type <id|cid>`: Interpret --cid as 'id' or 'cid' (default: 'id')
- `--signer-address <address>`: Signer address (0x...) overrides env
- `--signer-private-key <key>`: Signer private key (iotaprivkey1...); if omitted you will be prompted to insert it
- `--gas-id <coin-id>`: Gas coin object id (0x...) overrides env
- `--debug`: Verbose debug

#### 3.5.4. List Open Offers

```bash
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

```bash
go run main.go iota_sc withdraw --cid <cid> --idx <index> [--cid-type <id|cid>] [--signer-address <address>] [--signer-private-key <key>] [--gas-id <coin-id>] [--debug]
```

**Description:**

- Withdraw IOTA from a fulfilled offer

**Options:**

- `--cid <cid>`: CID (object id 0x... or CID string) (required)
- `--idx <index>`: Payment index to withdraw (0-based) (required)
- `--cid-type <id|cid>`: Interpret --cid as 'id' or 'cid' (default: 'id')
- `--signer-address <address>`: Signer address (0x...) overrides env
- `--signer-private-key <key>`: Signer private key (iotaprivkey1... / suiprivkey1... or base64 keystore); if omitted, you will be prompted
- `--gas-id <coin-id>`: Gas coin object id (0x...) overrides env
- `--debug`: Verbose debug

## 4. Utils from Iota CLI

Utility commands for inspecting and monitoring the system state using the IOTA client CLI.

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
iota client object 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b --json
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

<!-- TODO add the following -->
iota client faucet --address 0x731f57d2f3c5b102e5a4d182c8d4b6c06f0ba3aaf5534e1913e99631163d3edd

iota keytool import iotaprivkey1qp5n5ermut5gvfnvcdp5yhdk50jkdurqydyu7pz3e5qysdvpe6xtqka84au ed25519 --alias dcs_cid_owner

iota client switch --address 0x731f57d2f3c5b102e5a4d182c8d4b6c06f0ba3aaf5534e1913e99631163d3edd

iota client addresses

## 5. Main Test Scenario

This section outlines a complete test scenario for the CID lifecycle, from creation through offer submission, approval, honoring, and withdrawal. In this scenario we provide the commands for an object with CID `0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b`. Furthermore, epoch duration has been adjusted as following to ease the tests:

- during the first 10 minutes of each epoch, an offer can be submitted by the user
- after an offer has been approved, the user should honor it during the succeeding epoch (10 minutes before the epoch transition could be performed)

### 1. Create CID (user)

```bash
go run main.go iota_sc cid create QmaQXHJTDFKkcgKaMBRLpG9pobg2a5ph44kF5Z8u2UU2iG --epoch-start 0 --epoch-end 0 --type cid
```

### 2. Submit Offer (provider)

```bash
go run main.go iota_sc submit_offer --cid 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b --amount 100000 --cid-type id
```

**Notes:**

- Must be executed within the time limit

### 3. List Open Offers

```bash
go run main.go iota_sc list-open-offers
```

**Notes:**

- Shows active offers and the time remaining to approve them

### 4. Approve Offer (user)

```bash
go run main.go iota_sc approve_offer --cid 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b --cid-type id --idx 0
```

**Notes:**

- Execute immediately after the time limit for submitting offers has expired
- The offer is in the current epoch, with `approved: true`, `honored: false`, `paid: false`

### 5. Transition to Next Epoch (user)

```bash
go run main.go iota_sc cid next-epoch 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b --cid-type id
```

**Notes:**

- Execute immediately after the previous step
- Only effective after the actual end of the current epoch
- The offer is in the previous epoch, with `approved: true`, `honored: true`, `paid: false`

### 6. Honor Offer (user)

```bash
go run main.go iota_sc honor_offer --cid 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b --idx 0
```

**Notes:**

- Must be executed before the end of the current epoch
- The offer is in the current epoch, with `approved: true`, `honored: true`, `paid: false`

### 7. Transition to Next Epoch (Again) (user)

```bash
go run main.go iota_sc cid next-epoch 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b --cid-type id
```

**Notes:**

- Only effective after the actual end of the current epoch
- The offer is in the previous epoch, with `approved: true`, `honored: true`, `paid: false`

### 8. Withdraw (provider)

```bash
go run main.go iota_sc withdraw --cid 0x7593935b40a3fa1a5920999bf531bbfaac6f7dd63c86599962a241c286aa592b --cid-type id --idx 0
```

**Notes:**

- Execute after the ex-current epoch (now previous epoch) has ended and before transitioning to the next one
- The offer is in the previous epoch, with `approved: true`, `honored: true`, `paid: true`
