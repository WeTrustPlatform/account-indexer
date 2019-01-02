# account-indexer
An indexer to get all transactions of an account.

## Design
TODO: insert an image

## Database
We use LevelDB key-value database to store all of the data, this is similar to geth.

## Address database
Given an address, we can get all records with ${address} prefix in key.
${address}${block}${sequence}=${tx_hash}${other_address}${time}${value}

## Batch Status database
This is to track the sync status of batch process to sync from block genesis to latest
${incremental_id}=${time}${block_number}

## Block database
This is used by the "newHead" subscribe to handle Reorg scenario.
${block_number}=${address_1}${seq_1}...${address_n}${seq_n}

## Handle Restart
+ If latest block in Block_Status database < First block in Block database -> run batch again for missing blocks
+ Run batch from latest block in Block database until the current (latest) block
+ Start newHead subscription to geth