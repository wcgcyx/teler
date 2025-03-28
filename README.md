# Teler

Teler Node is a ***T***rust-based ***E***ssential-data ***L***ightweight ***E***thereum ***R***PC Node written in Golang.

# Introduction
Running an Ethereum RPC Node locally is expensive. Currently, a full Ethereum Node on the Mainnet typically requires over 1 TB of disk space.

As a result, developers often rely on external node providers, like [Alchemy](https://www.alchemy.com/), [QuickNode](https://www.quicknode.com/), and [Infura](https://www.infura.io/), to query on-chain data or perform transaction simulations.

The Teler Node offers a cost-effective alternative by significantly reducing disk usage; it retains chain state data only for the past hour and fetches the latest block from a trusted source. Once synced, a Teler Node stays updated by querying the latest block from a node provider, which can easily be managed through a free-tier subscription.

This setup enables Teler to serve RPC calls that target chain state within the past hour, including `eth_call`, `eth_estimateGas`, and tracing APIs.

Additionally, Teler can serve as a scalable solution for Web3 providers. Instead of deploying additional full Ethereum Nodes, cost-efficient Teler Nodes can be spun up and connected to an existing Ethereum Node to maintain sync, reducing both cost and complexity.

_Note: This documentation and teler is still WIP, please do NOT use teler for any production purpose before the first release is out. Feel free to reach out for any question._

# Datadir size
The following table summarises the disk usage of a synced Teler compared with Erigon (with `--prune=hrtc`).
|          Network      |   Teler   |   Erigon   |
| :-------------------: | :-------: |  :-------: |
|  Mainnet (1/Feb/2025) |   244G    |    1518G   |
|  Sepolia (1/Feb/2025) |   74G     |    680G    |
|  Holesky (1/Feb/2025) |   42G     |    262G    |

# Usage
## Build
Building `teler` requires Go >= 1.22.6. To obtain Go, visit [here](https://go.dev/doc/install).
```
git clone https://github.com/wcgcyx/teler.git
cd teler
make
./build/teler --help
```
## Quickstart (Holesky)
To sync a `teler` node on holesky from scratch, it is recommended to do initial sync from a local Ethereum node first before switching to a node provider. [Erigon](https://github.com/erigontech/erigon) is used below as an example.

(If you wish to sync from a snapshot directly to save time, reach out to me).

1. Spin up an Erigon node for initial sync
```
mkdir erigondir
erigon --internalcl --prune=hrtc --datadir ./erigondir --chain=holesky
```

2. Start teler node
```
mkdir telerdir
teler start --path ./telerdir --chain holesky --blksrc http://localhost:8545
```

3. Switch to node provider and start serving RPCs

Once teler syncs to the local erigon. You can stop both processes and remove everything erigon related to save disk space.

Then you will be able to sync from a node provider (a free tier subscription will be more than sufficient).

```
teler start --path ./telerdir --blksrc ${HOLESKY_PROVIDER_URL} --rpc-port 9424
```

This also serves RPC on `localhost:9424`.

# Supported RPC Methods
```
# Ethereum methods
eth_blobBaseFee
eth_blockNumber
eth_call
eth_chainId
eth_estimateGas
eth_feeHistory
eth_gasPrice
eth_getBalance
eth_getBlockByHash
eth_getBlockByNumber
eth_getBlockReceipts
eth_getBlockTransactionCountByHash
eth_getBlockTransactionCountByNumber
eth_getCode
eth_getLogs
eth_getStorageAt
eth_getTransactionByBlockHashAndIndex
eth_getTransactionByBlockNumberAndIndex
eth_getTransactionByHash
eth_getTransactionCount
eth_getTransactionReceipt
eth_getUncleByBlockHashAndIndex
eth_getUncleByBlockNumberAndIndex
eth_getUncleCountByBlockHash
eth_getUncleCountByBlockNumber
eth_maxPriorityFeePerGas

# Trace methods
trace_blockByNumber
trace_blockByHash
trace_transaction
trace_call
```
More in progress.

# Contributor
Zhenyang Shi - wcgcyx@gmail.com

# License
Licensed under LGPL-3.0.