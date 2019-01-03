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
		BlockNumber: "3000000",
		Addresses: []types.AddressSequence{
			types.AddressSequence{Address: address1, Sequence: 1},
			types.AddressSequence{Address: address2, Sequence: 2},
		},
	}
	encoded := bm.MarshallBlockDBValue(blockIndex)
	if len(encoded) != (gethcommon.AddressLength+1)*2 {
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

	for i, address := range addresses {
		if !strings.EqualFold(address.Address, blockIndex.Addresses[i].Address) {
			t.Error("Encoded address is not correct")
		}
		if address.Sequence != blockIndex.Addresses[i].Sequence {
			t.Error("Sequence is not correct")
		}
	}
}

func TestByteMarshallAddressKey(t *testing.T) {
	bm := ByteMarshaller{}
	address := "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e"
	blockNumber := "3000000"
	sequence := uint8(1)
	addressKey := bm.MarshallAddressKeyStr(address, blockNumber, sequence)
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

func TestMarshallBatchValue(t *testing.T) {
	bm := ByteMarshaller{}
	updatedAt := big.NewInt(time.Now().UnixNano())
	currentBlock := big.NewInt(3000000)
	value := bm.MarshallBatchValue(updatedAt, currentBlock)
	batchStatus := bm.UnmarshallBatchValue(value)
	if batchStatus.UpdatedAt.Cmp(updatedAt) != 0 {
		t.Error("UpdatedAt is not cortrect")
	}
	if batchStatus.Current.Cmp(currentBlock) != 0 {
		t.Error("Current block is not correct")
	}
}

func TestMarshallBatchKey(t *testing.T) {
	bm := ByteMarshaller{}
	from := big.NewInt(2)
	to := big.NewInt(100000)
	key := bm.MarshallBatchKey(from, to)
	batchStatus := bm.UnmarshallBatchKey(key)
	if batchStatus.From.Cmp(from) != 0 {
		t.Error("From is not correct")
	}
	if batchStatus.To.Cmp(to) != 0 {
		t.Error("To is not correct")
	}
}

func TestMarshallBlockKey(t *testing.T) {
	bm := ByteMarshaller{}
	blockNumberStr := "3000000"
	key := bm.MarshallBlockKey(blockNumberStr)
	blockNumber := bm.UnmarshallBlockKey(key)
	if blockNumber.String() != blockNumberStr {
		t.Error("Marshall/Unmarshall for block key is not correct")
	}
}
