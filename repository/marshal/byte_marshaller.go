package marshal

import (
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	TIMESTAMP_BYTE_LENGTH = 8
)

// ByteMarshaller marshal data using byte array
type ByteMarshaller struct {
}

// MarshallBlockDBValue marshall a blockIndex to []byte so that we store it as value in Block db
func (bm ByteMarshaller) MarshallBlockDBValue(blockIndex types.BlockIndex) []byte {
	length := len(blockIndex.Addresses)
	result := make([]byte, length*gethcommon.AddressLength)
	for i, address := range blockIndex.Addresses {
		addressByteArr, _ := hexutil.Decode(address)
		for j, byteItem := range addressByteArr {
			result[i*gethcommon.AddressLength+j] = byteItem
		}
	}
	return result
}

// UnmarshallBlockDBValue unmarshall a byte array into array of address, this is for Block db
func (bm ByteMarshaller) UnmarshallBlockDBValue(value []byte) []string {
	result := []string{}
	tmp := make([]byte, gethcommon.AddressLength)
	for i, byteItem := range value {
		if i > 0 && (i%gethcommon.AddressLength == 0) {
			address := hexutil.Encode(tmp)
			result = append(result, address)
		}
		tmp[i%gethcommon.AddressLength] = byteItem
	}
	if len(value) > 0 {
		address := hexutil.Encode(tmp)
		result = append(result, address)
	}

	return result
}

// MarshallAddressKey create LevelDB key
func (bm ByteMarshaller) MarshallAddressKey(index types.AddressIndex) []byte {
	return bm.MarshallAddressKeyStr(index.Address, index.BlockNumber.String())
}

// MarshallAddressKeyStr create LevelDB key
func (bm ByteMarshaller) MarshallAddressKeyStr(address string, blockNumber string) []byte {
	blockNumberBI := new(big.Int)
	blockNumberBI.SetString(blockNumber, 10)
	// 20 bytes
	resultByteArr, _ := hexutil.Decode(address)
	blockNumberByteArr := blockNumberBI.Bytes()
	result := append(resultByteArr, blockNumberByteArr...)
	return result
}

// MarshallAddressValue create LevelDB value
func (bm ByteMarshaller) MarshallAddressValue(index types.AddressIndex) []byte {
	// 32 byte
	txHashByteArr, _ := hexutil.Decode(index.TxHash)
	// 20 byte
	addressByteArr, _ := hexutil.Decode(index.CoupleAddress)
	// 8 byte
	timeByteArr := index.Time.Bytes()
	valueByteArr := []byte(index.Value.String())
	result := append(txHashByteArr, addressByteArr...)
	result = append(result, timeByteArr...)
	result = append(result, valueByteArr...)
	return result
}

// UnmarshallAddressKey LevelDB key to address_blockNumber
func (bm ByteMarshaller) UnmarshallAddressKey(key []byte) (string, *big.Int) {
	address := hexutil.Encode(key[:gethcommon.AddressLength])
	blockNumberBI := new(big.Int)
	blockNumberBI.SetBytes(key[gethcommon.AddressLength:])
	return address, blockNumberBI
}

// UnmarshallAddressValue LevelDB value to txhash_Value_Time
func (bm ByteMarshaller) UnmarshallAddressValue(value []byte) types.AddressIndex {
	hashLength := gethcommon.HashLength
	addressLength := gethcommon.AddressLength
	txHash := hexutil.Encode(value[:hashLength])
	address := hexutil.Encode(value[hashLength : hashLength+addressLength])
	timestamp := new(big.Int)
	timestamp.SetBytes(value[hashLength+addressLength : hashLength+addressLength+TIMESTAMP_BYTE_LENGTH])
	txValue := string(value[hashLength+addressLength+TIMESTAMP_BYTE_LENGTH:])
	txValueBI := new(big.Int)
	txValueBI.SetString(txValue, 10)
	result := types.AddressIndex{
		TxHash:        txHash,
		CoupleAddress: address,
		Time:          *timestamp,
		Value:         *txValueBI,
	}
	return result
}
