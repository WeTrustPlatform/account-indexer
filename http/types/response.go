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
