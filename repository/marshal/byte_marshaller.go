package marshal

import (
	"math/big"
	"strings"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	TIMESTAMP_BYTE_LENGTH = 4
)

// ByteMarshaller marshal data using byte array
type ByteMarshaller struct {
}

// MarshallBlockDBValue marshall a blockIndex to []byte so that we store it as value in Block db
func (bm ByteMarshaller) MarshallBlockDBValue(blockIndex types.BlockIndex) []byte {
	length := len(blockIndex.Addresses)
	// address1_seq1_address2_seq2
	result := make([]byte, length*(gethcommon.AddressLength+1))
	for i, addressSeq := range blockIndex.Addresses {
		address := addressSeq.Address
		addressByteArr, _ := hexutil.Decode(address)
		for j, byteItem := range addressByteArr {
			result[i*(gethcommon.AddressLength+1)+j] = byteItem
		}
		// Last byte is the sequence
		result[i*(gethcommon.AddressLength+1)+gethcommon.AddressLength] = addressSeq.Sequence
	}
	return result
}

// UnmarshallBlockDBValue unmarshall a byte array into array of address, this is for Block db
func (bm ByteMarshaller) UnmarshallBlockDBValue(value []byte) []types.AddressSequence {
	result := []types.AddressSequence{}
	// tmp := make([]byte, gethcommon.AddressLength)
	addressSeqLen := gethcommon.AddressLength + 1

	numAddress := len(value) / (addressSeqLen)
	for i := 0; i < numAddress; i++ {
		address := hexutil.Encode(value[i*addressSeqLen : (i+1)*addressSeqLen-1])
		sequence := value[(i+1)*addressSeqLen-1]
		addressSequence := types.AddressSequence{Address: address, Sequence: sequence}
		result = append(result, addressSequence)
	}

	return result
}

// MarshallAddressKey create LevelDB key
func (bm ByteMarshaller) MarshallAddressKey(index types.AddressIndex) []byte {
	return bm.MarshallAddressKeyStr(index.Address, index.BlockNumber.String(), index.Sequence)
}

// MarshallAddressKeyStr create LevelDB key
func (bm ByteMarshaller) MarshallAddressKeyStr(address string, blockNumber string, sequence uint8) []byte {
	blockNumberBI := new(big.Int)
	blockNumberBI.SetString(blockNumber, 10)
	// 20 bytes
	resultByteArr, _ := hexutil.Decode(address)
	// 1 byte for sequence
	result := append(resultByteArr, sequence)
	blockNumberByteArr := blockNumberBI.Bytes()
	result = append(result, blockNumberByteArr...)
	return result
}

func (bm ByteMarshaller) MarshallAddressKeyPrefix(address string) []byte {
	resultByteArr, _ := hexutil.Decode(address)
	return resultByteArr
}

// MarshallAddressValue create LevelDB value
func (bm ByteMarshaller) MarshallAddressValue(index types.AddressIndex) []byte {
	// 32 byte
	txHashByteArr, _ := hexutil.Decode(index.TxHash)
	// 20 byte
	addressByteArr, _ := hexutil.Decode(index.CoupleAddress)
	// 4 byte
	timeByteArr := common.MarshallTime(&index.Time)
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
	// TODO: should we return sequence?
	blockNumberBI.SetBytes(key[gethcommon.AddressLength+1:])
	return address, blockNumberBI
}

// UnmarshallAddressValue LevelDB value to txhash_Value_Time
func (bm ByteMarshaller) UnmarshallAddressValue(value []byte) types.AddressIndex {
	hashLength := gethcommon.HashLength
	addressLength := gethcommon.AddressLength
	txHash := hexutil.Encode(value[:hashLength])
	address := hexutil.Encode(value[hashLength : hashLength+addressLength])
	timestamp := common.UnmarshallTimeToInt(value[hashLength+addressLength : hashLength+addressLength+TIMESTAMP_BYTE_LENGTH])
	txValueBI := new(big.Int)
	txValue := string(value[hashLength+addressLength+TIMESTAMP_BYTE_LENGTH:])

	txValueBI.SetString(txValue, 10)
	result := types.AddressIndex{
		TxHash:        txHash,
		CoupleAddress: address,
		Time:          *timestamp,
		Value:         *txValueBI,
	}
	return result
}

// MarshallBatchValue value of key-value init batch status database
func (bm ByteMarshaller) MarshallBatchValue(updatedAt *big.Int, currentBlock *big.Int) []byte {
	// 4 byte
	timeByteArr := common.MarshallTime(updatedAt)
	blockNumberByteArr := currentBlock.Bytes()
	result := append(timeByteArr, blockNumberByteArr...)
	return result
}

// UnmarshallBatchValue unmarshal value of key-value init batch status database
func (bm ByteMarshaller) UnmarshallBatchValue(value []byte) types.BatchStatus {
	timestamp := common.UnmarshallTimeToInt(value[:TIMESTAMP_BYTE_LENGTH])
	currentBlock := new(big.Int)
	currentBlock.SetBytes(value[TIMESTAMP_BYTE_LENGTH:])
	return types.BatchStatus{
		UpdatedAt: timestamp,
		Current:   currentBlock,
	}
}

func (bm ByteMarshaller) MarshallBatchKey(from *big.Int, to *big.Int) []byte {
	fromStr := blockNumberWidPad(from.String())
	toStr := blockNumberWidPad(to.String())
	keyStr := fromStr + "_" + toStr
	return []byte(keyStr)
}

func (bm ByteMarshaller) UnmarshallBatchKey(key []byte) types.BatchStatus {
	keyStr := string(key)
	keyArr := strings.Split(keyStr, "_")
	// TODO: handle error
	fromStr := keyArr[0]
	toStr := keyArr[1]
	from := new(big.Int)
	from.SetString(fromStr, 10)
	to := new(big.Int)
	to.SetString(toStr, 10)
	return types.BatchStatus{From: from, To: to}
}

func (bm ByteMarshaller) MarshallBlockKey(blockNumber string) []byte {
	return []byte(blockNumber)
}

func (bm ByteMarshaller) UnmarshallBlockKey(key []byte) *big.Int {
	blockNumber := string(key)
	result := new(big.Int)
	result.SetString(blockNumber, 10)
	return result
}
