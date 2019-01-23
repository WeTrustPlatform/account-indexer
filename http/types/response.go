package types

import (
	"math/big"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/core/types"
	coreTypes "github.com/WeTrustPlatform/account-indexer/core/types"
)

// EITotalTransaction total transaction of an account
type EITotalTransaction struct {
	Total int `json:"numFound"`
}

// EITransactionsByAccount response for getTransactionsByAccount api
type EITransactionsByAccount struct {
	Total   string      `json:"numFound"`
	Start   int         `json:"start"`
	Indexes []EIAddress `json:"data"`
}

// EIAddress response for getTransactionsByAccount api
type EIAddress struct {
	// coreTypes.AddressIndex
	Address string   `json:"address"`
	TxHash  string   `json:"txHash"`
	Value   *big.Int `json:"value"`
	Time    string   `json:"time"`
	// BlockNumber   *big.Int `json:"blockNumber"`
	CoupleAddress string   `json:"coupleAddress"`
	Data          []byte   `json:"data"`
	Gas           uint64   `json:"gas"`
	GasPrice      *big.Int `json:"gasPrice"`
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

// AddressToEIAddress business data type to EI data type
func AddressToEIAddress(address types.AddressIndex) EIAddress {
	return EIAddress{
		Address:       address.Address,
		TxHash:        address.TxHash,
		Value:         address.Value,
		Time:          common.UnmarshallIntToTime(address.Time).Format(time.RFC3339),
		CoupleAddress: address.CoupleAddress,
	}
}
