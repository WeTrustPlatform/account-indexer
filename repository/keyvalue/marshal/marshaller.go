package marshal

import (
	"math/big"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

// Marshaller the interface to convert business objects to/from byte
type Marshaller interface {
	MarshallBatchValue(updatedAt *big.Int, currentBlock *big.Int) []byte
	UnmarshallBatchValue(value []byte) types.BatchStatus
	MarshallBatchKey(from *big.Int, to *big.Int, step byte, createdAt *big.Int) []byte
	MarshallBatchKeyFrom(from *big.Int) []byte
	UnmarshallBatchKey(value []byte) types.BatchStatus
	MarshallBlockValue(blockIndex *types.BlockIndex) []byte
	UnmarshallBlockValue(value []byte) types.BlockIndex
	MarshallBlockKey(blockNumber string) []byte
	UnmarshallBlockKey(key []byte) *big.Int
	MarshallAddressKey(index *types.AddressIndex) []byte
	MarshallAddressKeyPrefix(address string) []byte
	MarshallAddressKeyPrefix2(address string, time *big.Int) []byte
	MarshallAddressKeyPrefix3(address string, tm time.Time) []byte
	MarshallAddressKeyStr(address string, time *big.Int, sequence uint8) []byte
	MarshallAddressValue(index *types.AddressIndex) []byte
	UnmarshallAddressKey(key []byte) (string, *big.Int)
	UnmarshallAddressValue(value []byte) types.AddressIndex
}
