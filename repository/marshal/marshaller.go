package marshal

import (
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

type Marshaller interface {
	MarshallBatchValue(updatedAt *big.Int, currentBlock *big.Int) []byte
	UnmarshallBatchValue(value []byte) types.BatchStatus
	MarshallBatchKey(from *big.Int, to *big.Int) []byte
	UnmarshallBatchKey(value []byte) types.BatchStatus
	MarshallBlockDBValue(blockIndex types.BlockIndex) []byte
	UnmarshallBlockDBValue(value []byte) []types.AddressSequence
	MarshallBlockKey(blockNumber string) []byte
	UnmarshallBlockKey(key []byte) *big.Int
	MarshallAddressKey(index types.AddressIndex) []byte
	MarshallAddressKeyStr(address string, blockNumber string, sequence uint8) []byte
	MarshallAddressValue(index types.AddressIndex) []byte
	UnmarshallAddressKey(key []byte) (string, *big.Int)
	UnmarshallAddressValue(value []byte) types.AddressIndex
}
