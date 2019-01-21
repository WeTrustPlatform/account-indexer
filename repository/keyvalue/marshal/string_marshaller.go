package marshal

import (
	"math/big"
	"strings"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

// StringMarshaller initial marshaller impl
type StringMarshaller struct {
}

// MarshallBlockDBValue marshall a blockIndex to []byte so that we store it as value in Block db
func (sm StringMarshaller) MarshallBlockDBValue(blockIndex *types.BlockIndex) []byte {
	addresses := []string{}
	for _, item := range blockIndex.Addresses {
		addresses = append(addresses, item.Address)
	}
	value := strings.Join(addresses, "_")
	return []byte(value)
}

// UnmarshallBlockDBValue unmarshall a byte array into array of address, this is for Block db
func (sm StringMarshaller) UnmarshallBlockDBValue(value []byte) []string {
	valueStr := string(value)
	valueArr := strings.Split(valueStr, "_")
	return valueArr
}

// MarshallAddressKey create LevelDB key
func (sm StringMarshaller) MarshallAddressKey(index *types.AddressIndex) []byte {
	// return sm.MarshallAddressKeyStr(index.Address, index.BlockNumber.String())
	return sm.MarshallAddressKeyStr(index.Address, "")
}

// MarshallAddressKeyStr create LevelDB key
func (sm StringMarshaller) MarshallAddressKeyStr(address string, blockNumber string) []byte {
	key := address + "_" + blockNumberWidPad(blockNumber)
	key = strings.ToUpper(key)
	return []byte(key)
}

// MarshallAddressValue create LevelDB value
func (sm StringMarshaller) MarshallAddressValue(index *types.AddressIndex) []byte {
	value := index.TxHash + "_" + index.Value.String() + "_" + index.Time.String()
	return []byte(value)
}

// UnmarshallAddressKey LevelDB key to address_blockNumber
func (sm StringMarshaller) UnmarshallAddressKey(key []byte) (string, *big.Int) {
	keyStr := string(key)
	keyArr := strings.Split(keyStr, "_")
	blockNumber := stringToBigInt(keyArr[1])
	return keyArr[0], blockNumber
}

// UnmarshallAddressValue LevelDB value to txhash_Value_Time
func (sm StringMarshaller) UnmarshallAddressValue(value []byte) types.AddressIndex {
	valueStr := string(value)
	valueArr := strings.Split(valueStr, "_")
	txHash := valueArr[0]
	txValue := stringToBigInt(valueArr[1])
	time := stringToBigInt(valueArr[2])
	return types.AddressIndex{
		TxHash: txHash,
		Value:  txValue,
		Time:   time,
	}
}

func stringToBigInt(str string) *big.Int {
	result := new(big.Int)
	result, _ = result.SetString(str, 10)
	return result
}
