package indexer

import (
	"math/big"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

// TODO: test using memdb https://godoc.org/github.com/syndtr/goleveldb/leveldb/memdb#New

var now = *big.NewInt(time.Now().Unix())

var index = types.AddressIndex{
	Address:     "0xF58b12474c084B3Bcd32B991ea1BABdf0d67c109",
	TxHash:      "0xaf27ec30685cbb8acb995051825b7651801beb3101c5d62d0ae00e78a2801543",
	Value:       *big.NewInt(100),
	Time:        now,
	BlockNumber: *big.NewInt(2018),
}

func TestMarshallKey(t *testing.T) {
	addressKeyByte := MarshallKey(index)
	// fmt.Println(len(addressKeyByte))
	addressKeyStr := string(addressKeyByte)
	if addressKeyStr != "0XF58B12474C084B3BCD32B991EA1BABDF0D67C109_0000002018" {
		t.Error("Address key is not correct")
	}
}

func TestMarshallValue(t *testing.T) {
	txDataStr := string(MarshallValue(index))
	if txDataStr != ("0xaf27ec30685cbb8acb995051825b7651801beb3101c5d62d0ae00e78a2801543_100_" + now.String()) {
		t.Error("Transaction value is not correct")
	}
}

func TestUnmashallKey(t *testing.T) {
	byteArr := []byte("0XF58B12474C084B3BCD32B991EA1BABDF0D67C109_0000002018")
	address, blockNumber := UnmarshallKey(byteArr)
	if address != "0XF58B12474C084B3BCD32B991EA1BABDF0D67C109" {
		t.Error("Unmarshalled address is not correct")
	}
	if blockNumber.String() != "2018" {
		t.Error("Unmarshalled block number is not correct")
	}
}

func TestUnmarshallValue(t *testing.T) {
	byteArr := []byte("0xaf27ec30685cbb8acb995051825b7651801beb3101c5d62d0ae00e78a2801543_100_" + now.String())
	addressIndex := UnmarshallValue(byteArr)
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

// TODO: unit test for block DB
// TODO: unit test for reorg scenario
