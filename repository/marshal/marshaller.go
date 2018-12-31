package marshal

import (
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

type Marshaller interface {
	MarshallBlockDBValue(blockIndex types.BlockIndex) []byte
	UnmarshallBlockDBValue(value []byte) []string
	MarshallAddressKey(index types.AddressIndex) []byte
	MarshallAddressKeyStr(address string, blockNumber string) []byte
	MarshallAddressValue(index types.AddressIndex) []byte
	UnmarshallAddressKey(key []byte) (string, *big.Int)
	UnmarshallAddressValue(value []byte) types.AddressIndex
}
