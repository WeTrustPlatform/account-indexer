# account-indexer
An indexer to get all transactions of an account.

## Design
TODO: insert an image

## Database
We use LevelDB key-value database to store all of the data, this is similar to geth.

## Address database
Given an address, we can get all records with ${address} prefix in key.
${address}${time}${sequence}=${tx_hash}${other_address}${blockNumber}${value}

## Batch Status database
This is to track the sync status of batch process
Initially, a batch has "from" as genesis block and "to" as latest block.
A batch can be from the last newHead block in DB to the latest block in block chain
${from_to}=${time}${block_number}

## Block database
This is used by the "newHead" subscribe to handle Reorg scenario.
${block_number}=${address_1}${seq_1}...${address_n}${seq_n}

## Handle Reorg
If an old block comes again, get time and address sequences from block database, delete respective records from address database

## Handle Restart
+ If there is no batch status database, run from genesis to latest as a batch, track status at each block processed
+ Else if there are any uncompleted batches, run all of them. And run a separate batch from the last newHead block to the latest block in blockchain
+ Start newHead subscription to geth