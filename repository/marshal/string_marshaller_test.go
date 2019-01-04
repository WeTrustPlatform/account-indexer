package marshal

import (
	"math/big"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

var now = *big.NewInt(time.Now().Unix())

var index = types.AddressIndex{
	AddressSequence: types.AddressSequence{
		Address:  "0xF58b12474c084B3Bcd32B991ea1BABdf0d67c109",
		Sequence: 1,
	},
	TxHash:      "0xaf27ec30685cbb8acb995051825b7651801beb3101c5d62d0ae00e78a2801543",
	Value:       *big.NewInt(100),
	Time:        now,
	BlockNumber: *big.NewInt(2018),
}

var marshaller = StringMarshaller{}

func TestMarshallKey(t *testing.T) {
	addressKeyByte := marshaller.MarshallAddressKey(index)
	addressKeyStr := string(addressKeyByte)
	if addressKeyStr != "0XF58B12474C084B3BCD32B991EA1BABDF0D67C109_0000002018" {
		t.Error("Address key is not correct")
	}
}

func TestMarshallValue(t *testing.T) {
	txDataStr := string(marshaller.MarshallAddressValue(index))
	if txDataStr != ("0xaf27ec30685cbb8acb995051825b7651801beb3101c5d62d0ae00e78a2801543_100_" + now.String()) {
		t.Error("Transaction value is not correct")
	}
}

func TestUnmashallKey(t *testing.T) {
	byteArr := []byte("0XF58B12474C084B3BCD32B991EA1BABDF0D67C109_0000002018")
	address, blockNumber := marshaller.UnmarshallAddressKey(byteArr)
	if address != "0XF58B12474C084B3BCD32B991EA1BABDF0D67C109" {
		t.Error("Unmarshalled address is not correct")
	}
	if blockNumber.String() != "2018" {
		t.Error("Unmarshalled block number is not correct")
	}
}

func TestUnmarshallValue(t *testing.T) {
	byteArr := []byte("0xaf27ec30685cbb8acb995051825b7651801beb3101c5d62d0ae00e78a2801543_100_" + now.String())
	addressIndex := marshaller.UnmarshallAddressValue(byteArr)
	if addressIndex.TxHash != "0xaf27ec30685cbb8acb995051825b7651801beb3101c5d62d0ae00e78a2801543" {
		t.Error("Unmarshalled transaction hash is not correct")
	}

	if addressIndex.Value.String() != "100" {
		t.Error("Unmarshalled transaction value is not correct")
	}

	if addressIndex.Time.String() != now.String() {
		t.Error("Unmarshalled transaction time is not correct")
	}
}
