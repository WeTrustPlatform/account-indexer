package types

import (
	"math/big"
	"time"

	coreTypes "github.com/WeTrustPlatform/account-indexer/core/types"
)

// EITransactionsByAccount response for getTransactionsByAccount api
type EITransactionsByAccount struct {
	Total   int                      `json:"numFound"`
	Start   int                      `json:"start"`
	Indexes []coreTypes.AddressIndex `json:"data"`
}

// EIBlocks list of blocks to return to frontend
type EIBlocks struct {
	Total   int                    `json:"numFound"`
	Start   int                    `json:"start"`
	Indexes []coreTypes.BlockIndex `json:"data"`
}

// EIBatchStatus batch status
type EIBatchStatus struct {
	From      *big.Int  `json:"from"`
	To        *big.Int  `json:"to"`
	Step      byte      `json:"Step"`
	Current   string    `json:"current"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
