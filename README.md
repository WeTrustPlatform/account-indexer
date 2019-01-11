# account-indexer
An indexer to get all transactions of an account.

## Design
TODO: insert an image

## Technology stack
We use the same technology stack like go-ethereum: golang - LevelDB. For REST, we use gin. We also leverage ethclient package of go-ethereum to connect with a geth node through ipc.

## Address database
Given an address, we can get all records with ${address} prefix in key.
${address}${block_time}${sequence}=${tx_hash}${other_address}${blockNumber}${value}
+ Search by address and time range is very performant
+ To handle reorg scenario, get address and block time from block database.

## Batch Status database
This is to track the sync status of batch process
Initially, a batch has "from" as genesis block and "to" as latest block.
A batch can be from the last newHead block in DB to the latest block in block chain
${from}${to}${created_at}=${updated_at}${current_block_number}

## Block database
This is used by the "newHead" subscribe to handle Reorg scenario.
${block_number}=${created_at}${block_time}${address_1}${seq_1}...${address_n}${seq_n}

## Handle Reorg
If an old block comes again, get time and address sequences from block database, delete respective records from address database

## Handle node out of sync
Use a go routine to regularly check for last record of block database amongst current time, created at and block time

## Handle block db
Block db is borned for Reorg scenario and we need last block most of the time.
Use a go routine to regularly delete old blocks.

## Handle Restart
+ If there is no batch status database, run from genesis to latest as a batch, track status at each block processed
+ Else if there are any uncompleted batches, run all of them, each in a separate go routine. And run a separate batch from the last newHead block to the latest block in blockchain
+ Start newHead subscription to geth