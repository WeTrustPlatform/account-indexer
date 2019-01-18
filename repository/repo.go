package repository

import (
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

// IndexRepo to store index data
type IndexRepo interface {
	Store(indexData []*types.AddressIndex, blockIndex *types.BlockIndex, isBatch bool) error
	GetTransactionByAddress(address string, rows int, start int, fromTime *big.Int, toTime *big.Int) (int, []types.AddressIndex)
	GetLastBlock() (types.BlockIndex, error)
	GetFirstBlock() (types.BlockIndex, error)
	DeleteOldBlocks(untilTime *big.Int) (int, error)
	GetBlocks(blockNumber string, rows int, start int) (int, []types.BlockIndex)
}

// BatchRepo repository for batch status
type BatchRepo interface {
	GetAllBatchStatuses() []types.BatchStatus
	UpdateBatch(batch types.BatchStatus) error
	ReplaceBatch(from *big.Int, newTo *big.Int) error
}
