package types

import (
	"math/big"
)

/**
 * This contains data that we received from blockchain
 */

// TransactionDetail to be indexed
type TransactionDetail struct {
	From   string
	To     string
	TxHash string
	Value  *big.Int
	Status bool
}

// TransactionExtra additional data to query geth node on the fly, this is not store in indexer DB
type TransactionExtra struct {
	Data     []byte
	Gas      uint64
	GasPrice *big.Int
}

// BLockDetail data received from blockchain
type BLockDetail struct {
	BlockNumber  *big.Int
	Time         *big.Int
	Transactions []TransactionDetail
}
