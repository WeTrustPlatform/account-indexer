package types

import (
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
