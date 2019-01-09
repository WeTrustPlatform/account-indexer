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
	AddressSequence
	TxHash        string   `json:"tx_hash"`
	Value         *big.Int `json:"value"`
	Time          *big.Int `json:"time"`
	BlockNumber   *big.Int `json:"blockNumber"`
	CoupleAddress string   `json:"coupleAddress"`
}

// AddressSequence In same block, 1 address can stay in multiple transactions, especially the "to"
type AddressSequence struct {
	Address  string `json:"address"`
	Sequence uint8  `json:"sequence"`
}

// BlockIndex index data for Block LevelDB
type BlockIndex struct {
	BlockNumber string
	Addresses   []AddressSequence
	Time        *big.Int
}

// InitBatchStatus the init batch status
type BatchStatus struct {
	// Block information for each batch
	From      *big.Int
	To        *big.Int
	Current   *big.Int
	UpdatedAt *big.Int
}

func (index AddressIndex) String() string {
	return fmt.Sprintf("address %s, tx hash: %s, value: %s, time: %v, block: %v", index.Address, index.TxHash, index.Value.String(), index.Time, index.BlockNumber)
}
