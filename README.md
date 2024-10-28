# Teler

Teler Node is a ***T***rust-based ***E***ssential-data ***L***ightweight ***E***thereum ***R***PC Node written in Golang.

# Introduction
It is expensive to run a Ethereum RPC Node locally. As of today, an Ethereum Full Node on Mainnet typically takes over 1 TB of disk space to run. 

Therefore, developers would need to rely on node providers to query on-chain data or perform transaction simulation such as [Alchemy](https://www.alchemy.com/), [QuickNode](https://www.quicknode.com/), [Infura](https://www.infura.io/) and so on. 

Teler node significantly reduces the disk usage by only keeping chain state for the past hour and by getting latest block from a trusted source. Once sync up, a teler node will be able to stay in sync by querying the latest block from a node provider, which can be easily achieved with a free tier subscription.

Meanwhile, it is able to serve RPCs that are targeting chain state < an hour. This includes `eth_call`, `eth_estimateGas` and even tracing APIs (_WIP_).

Teler node can also be used as a way to scale web3 provider. Instead of spining up new Ethereum Nodes to scale, much cheaper teler nodes can be spinned up and connecting to an existing Ethereum Node to stay in sync.

_Note: This documentation and teler is still WIP, feel free to reach out for any question._

# Usage
## Build
Building `teler` requires Go >= 1.22.5. To obtain Go, visit [here](https://go.dev/doc/install).
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

# Contributor
Zhenyang Shi - wcgcyx@gmail.com

# License
Licensed under LGPL-3.0.