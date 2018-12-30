package types

import (
	"fmt"
	"math/big"
)

/**
 * This contains data to be indexed
 */

// AddressIndex Transaction data of an address to be index
// Index data for Address LevelDB
// Value can be negative or positive
type AddressIndex struct {
	Address     string
	TxHash      string
	Value       big.Int
	Time        big.Int
	BlockNumber big.Int
}

// BlockIndex index data for Block LevelDB
type BlockIndex struct {
	BlockNumber string
	Addresses   []string
}

func (index AddressIndex) String() string {
	return fmt.Sprintf("address %s, tx hash: %s, value: %s, time: %v, block: %v", index.Address, index.TxHash, index.Value.String(), index.Time, index.BlockNumber)
}
