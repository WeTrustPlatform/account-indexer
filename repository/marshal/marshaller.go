package marshal

import (
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

type Marshaller interface {
	MarshallBlockDBValue(blockIndex types.BlockIndex) []byte
	UnmarshallBlockDBValue(value []byte) []types.AddressSequence
	MarshallAddressKey(index types.AddressIndex) []byte
	MarshallAddressKeyStr(address string, blockNumber string, sequence uint8) []byte
	MarshallAddressValue(index types.AddressIndex) []byte
	UnmarshallAddressKey(key []byte) (string, *big.Int)
	UnmarshallAddressValue(value []byte) types.AddressIndex
}
