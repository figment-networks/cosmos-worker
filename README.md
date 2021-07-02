# COSMOS WORKER

This repository contains a worker part dedicated for cosmos transactions.

## Worker
Stateless worker is responsible for connecting with the chain, getting information, converting it to a common format and sending it back to manager.
Worker can be connected with multiple managers but should always answer only to the one that sent request.

## API
Implementation of bare requests for network.

### Client
Worker's business logic wiring of messages to client's functions.


## Installation
This system can be put together in many different ways.
This readme will describe only the simplest one worker, one manager with embedded scheduler approach.

### Compile
To compile sources you need to have go 1.14.1+ installed.

```bash
    make build
```

### Running
Worker also need some basic config:

```bash
    MANAGERS=0.0.0.0:8085
    COSMOS_GRPC_ADDR=https://cosmoshub-4.node-address
    CHAIN_ID=cosmoshub-4
```

Where
    - `COSMOS_GRPC_ADDR` is a http address to a cosmos node's grpc endpoint
    - `MANAGERS` a comma-separated list of manager ip:port addresses that worker will connect to. In this case only one

After running both binaries worker should successfully register itself to the manager.

If you wanna connect with manager running on docker instance add `HOSTNAME=host.docker.internal` (this is for OSX and Windows). For linux add your docker gateway address taken from ifconfig (it probably be the one from interface called docker0).

## Developing Locally

First, you will need to set up a few dependencies:

1. [Install Go](https://golang.org/doc/install)
2. A Kava network node with both RPC and LCD APIs (in this example, we assume it's running at http://localhost)
3. A running [indexer-manager](https://github.com/figment-networks/indexer-manager) instance
4. A running datastore API instance (configured with `STORE_HTTP_ENDPOINTS`).

Then, run the worker with some environment config:

```
CHAIN_ID=cosmoshub-4 \
STORE_HTTP_ENDPOINTS=http://127.0.0.1:8986/input/jsonrpc \
COSMOS_GRPC_ADDR=0.0.0.0:9090 \
TENDERMINT_LCD_ADDR=http://127.0.0.1:1317 \
go run ./cmd/worker-cosmos
```

## Transaction Types
List of currently supported transaction types in cosmos-worker are (listed by modules):
- bank:
    `multisend` , `send`
- crisis:
    `verify_invariant`
- distribution:
    `withdraw_validator_commission` , `set_withdraw_address` , `withdraw_delegator_reward` , `fund_community_pool`
- evidence:
    `submit_evidence`
- gov:
    `deposit` , `vote` , `submit_proposal`
- slashing:
    `unjail`
- staking:
    `begin_unbonding` , `edit_validator` , `create_validator` , `delegate` , `begin_redelegate`
- vesting:
    `msg_create_vesting_account`
- internal:
    `error`

List of currently supported ibc transaction types in cosmos-worker are (listed by modules):
- channel:
    `channel_open_init` , `channel_open_confirm`, `channel_open_ack`, `channel_open_try`, `channel_close_init`, `channel_close_confirm`, `recv_packet`, `timeout`, `channel_acknowledgement`
- client:
    `create_client` , `update_client`, `upgrade_client`, `submit_misbehaviour`
- connection:
    `connection_open_init` , `connection_open_confirm`, `connection_open_ack`, `connection_open_try`
- transfer:
    `transfer`
- internal:
    `error`
