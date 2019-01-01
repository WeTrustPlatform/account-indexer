package marshal

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

func TestByteMarshallerBlock(t *testing.T) {
	bm := ByteMarshaller{}
	address1 := "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e"
	address2 := "0x7FA2B1C6E0B8B8805Bd56eC171aD8A8fbDEA3a44"
	blockIndex := types.BlockIndex{
		Addresses: []string{address1, address2},
	}
	encoded := bm.MarshallBlockDBValue(blockIndex)
	if len(encoded) != gethcommon.AddressLength*2 {
		t.Error("Encoded array length is not correct")
	}
	// 84 as in StringMarshaller
	stringByteLength := len([]byte(address1)) * 2
	// 40
	if (len(encoded)) > stringByteLength {
		t.Error("Encoded array length is not correct")
	}
	addresses := bm.UnmarshallBlockDBValue(encoded)
	if len(addresses) != 2 {
		t.Error("Decoded address array length is not correct")
	}
	if strings.ToUpper(addresses[0]) != strings.ToUpper(address1) {
		t.Error("Encoded address1 is not correct")
	}
	if strings.ToUpper(addresses[1]) != strings.ToUpper(address2) {
		t.Error("Encoded address2 is not correct")
	}
}

func TestByteMarshallAddressKey(t *testing.T) {
	bm := ByteMarshaller{}
	address := "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e"
	blockNumber := "3000000"
	addressKey := bm.MarshallAddressKeyStr(address, blockNumber)
	addressRst, blockNumberRst := bm.UnmarshallAddressKey(addressKey)
	if strings.ToUpper(address) != strings.ToUpper(addressRst) {
		t.Error("Unmarshalled address is wrong")
	}
	if blockNumberRst.String() != blockNumber {
		t.Error("Unmarshalled block number is not correct")
	}
}

func TestByteMarshallAddressValue(t *testing.T) {
	blockTime := big.NewInt(time.Now().UnixNano())
	blockNumber := big.NewInt(6000000)
	value := big.NewInt(1000000000)
	bm := ByteMarshaller{}
	addressIndex := types.AddressIndex{
		CoupleAddress: "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e",
		BlockNumber:   *blockNumber,
		TxHash:        "0x9bdbd233827534e48cc23801d145c64c4f4bab6b2c4c74a54673633e4c6c1591",
		Value:         *value,
		Time:          *blockTime,
	}
	indexValue := bm.MarshallAddressValue(addressIndex)
	addressIndex2 := bm.UnmarshallAddressValue(indexValue)
	// if !strings.EqualFold(addressIndex2.Address, addressIndex.Address) {
	// 	t.Error("Unmarshalled address is not correct")
	// }
	if !strings.EqualFold(addressIndex2.TxHash, addressIndex.TxHash) {
		t.Error("Unmarshalled tx hash is not correct")
	}
	if !strings.EqualFold(addressIndex2.CoupleAddress, addressIndex.CoupleAddress) {
		t.Error("Unmarshalled couple address is not correct")
	}
	if !strings.EqualFold(addressIndex2.Time.String(), addressIndex.Time.String()) {
		t.Error("Unmarshalled tx time is not correct")
	}
	if !strings.EqualFold(addressIndex2.Value.String(), addressIndex.Value.String()) {
		t.Error("Unmarshalled tx value is not correct")
	}
}
