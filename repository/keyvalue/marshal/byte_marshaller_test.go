package marshal

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestByteMarshallerBlock(t *testing.T) {
	bm := ByteMarshaller{}
	address1 := "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e"
	address2 := "0x7FA2B1C6E0B8B8805Bd56eC171aD8A8fbDEA3a44"
	blockTime := big.NewInt(time.Now().Unix())
	createdAt := blockTime
	blockIndex := &types.BlockIndex{
		BlockNumber: "3000000",
		Addresses: []types.AddressSequence{
			types.AddressSequence{Address: address1, Sequence: 1},
			types.AddressSequence{Address: address2, Sequence: 2},
		},
		Time:      blockTime,
		CreatedAt: createdAt,
	}
	encoded := bm.MarshallBlockValue(blockIndex)
	assert.Equal(t, 2*TimestampByteLength+(gethcommon.AddressLength+1)*2, len(encoded))
	// 40
	reBlockIndex := bm.UnmarshallBlockValue(encoded)
	assert.Equal(t, *blockTime, *reBlockIndex.Time)
	assert.Equal(t, *createdAt, *reBlockIndex.CreatedAt)
	assert.Equal(t, 2, len(reBlockIndex.Addresses))

	for i, address := range reBlockIndex.Addresses {
		assert.True(t, strings.EqualFold(address.Address, blockIndex.Addresses[i].Address))
		assert.Equal(t, address.Sequence, blockIndex.Addresses[i].Sequence)
	}
}

func TestByteMarshallAddressKey(t *testing.T) {
	bm := ByteMarshaller{}
	address := "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e"
	blockTime := big.NewInt(time.Now().Unix())
	sequence := uint8(1)
	addressKey := bm.MarshallAddressKeyStr(address, blockTime, sequence)
	addressRst, blockTimeRst := bm.UnmarshallAddressKey(addressKey)
	assert.Equal(t, strings.ToUpper(address), strings.ToUpper(addressRst))
	assert.Equal(t, blockTime, blockTimeRst)
}

func TestByteMarshallAddressKeyPrefix(t *testing.T) {
	bm := ByteMarshaller{}
	address := "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e"
	now := time.Now()
	tm2 := big.NewInt(now.Unix())
	byteArr2 := bm.MarshallAddressKeyPrefix2(address, tm2)
	byteArr3 := bm.MarshallAddressKeyPrefix3(address, now)
	assert.Equal(t, byteArr2, byteArr3)
}

func TestByteMarshallAddressValue(t *testing.T) {
	// blockNumber := big.NewInt(6000000)
	value := big.NewInt(1000000000)
	bm := ByteMarshaller{}
	addressIndex := &types.AddressIndex{
		CoupleAddress: "0xEcFf2b254c9354f3F73F6E64b9613Ad0a740a54e",
		// BlockNumber:   blockNumber,
		TxHash: "0x9bdbd233827534e48cc23801d145c64c4f4bab6b2c4c74a54673633e4c6c1591",
		Value:  value,
	}
	indexValue := bm.MarshallAddressValue(addressIndex)
	addressIndex2 := bm.UnmarshallAddressValue(indexValue)
	assert.True(t, strings.EqualFold(addressIndex2.TxHash, addressIndex.TxHash))
	assert.True(t, strings.EqualFold(addressIndex2.CoupleAddress, addressIndex.CoupleAddress))
	assert.True(t, strings.EqualFold(addressIndex2.Value.String(), addressIndex.Value.String()))
	// assert.Equal(t, blockNumber, addressIndex2.BlockNumber)
}

func TestMarshallBatchValue(t *testing.T) {
	bm := ByteMarshaller{}
	updatedAt := big.NewInt(time.Now().Unix())
	currentBlock := big.NewInt(3000000)
	value := bm.MarshallBatchValue(updatedAt, currentBlock)
	batchStatus := bm.UnmarshallBatchValue(value)
	assert.True(t, batchStatus.UpdatedAt.Cmp(updatedAt) == 0)
	assert.True(t, batchStatus.Current.Cmp(currentBlock) == 0)
}

func TestMarshallBatchKey(t *testing.T) {
	bm := ByteMarshaller{}
	from := big.NewInt(2)
	to := big.NewInt(100000)
	step := byte(3)
	createdAt := big.NewInt(time.Now().Unix())
	key := bm.MarshallBatchKey(from, to, step, createdAt)
	batchStatus := bm.UnmarshallBatchKey(key)
	assert.Equal(t, *from, *batchStatus.From)
	assert.Equal(t, *to, *batchStatus.To)
	assert.Equal(t, byte(3), batchStatus.Step)
	assert.Equal(t, *createdAt, *batchStatus.CreatedAt)
}

func TestMarshallBlockKey(t *testing.T) {
	bm := ByteMarshaller{}
	blockNumberStr := "3000000"
	key := bm.MarshallBlockKey(blockNumberStr)
	blockNumber := bm.UnmarshallBlockKey(key)
	assert.Equal(t, blockNumberStr, blockNumber.String())
}
