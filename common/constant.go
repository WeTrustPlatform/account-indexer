package common

const (
	// AddressZero some transactions do not have To address, use this address instead
	AddressZero = "0x0000000000000000000000000000000000000000"
	// NumMaxTransaction if number of transaction is more than this, just return +100000 to the client
	NumMaxTransaction = 100000
	// DefaultCleanInterval In minute
	DefaultCleanInterval = 5
	// DefaultBlockTTL In hour
	DefaultBlockTTL = 4
	// DefaultWatcherInterval in minute
	DefaultWatcherInterval = 5
	// DefaultOOSThreshold in minute
	DefaultOOSThreshold = 10
	// DefaultHTTPPort default http port
	DefaultHTTPPort = 3000
	// DefaultNumBatch default number of init batch
	DefaultNumBatch = 8
)
