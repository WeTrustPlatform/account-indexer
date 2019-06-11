package common

const (
	// AddressZero some transactions do not have To address, use this address instead
	AddressZero = "0x0000000000000000000000000000000000000000"
	// DefaultIpc the default ipc path
	DefaultIpc = "/home/blockform/.ethereum/geth.ipc"
	// DefaultDbPath default database path, a folder then prefix
	DefaultDbPath = "/home/blockform/account-indexer-db/geth_indexer_leveldb"
	// NumMaxTransaction if number of transaction is more than this, just return +10000 to the client
	NumMaxTransaction = 10000
	// DefaultCleanInterval In minute
	DefaultCleanInterval = 5
	// DefaultBlockTTL In hour
	DefaultBlockTTL = 4
	// DefaultWatcherInterval in minute
	DefaultWatcherInterval = 5
	// DefaultOOSThreshold in second
	DefaultOOSThreshold = 300
	// DefaultHTTPPort default http port
	DefaultHTTPPort = 3000
	// DefaultNumBatch default number of init batch
	DefaultNumBatch = 8
)
