package repository

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/dao"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
)

var blockTime = big.NewInt(time.Now().Unix())
var from1 = "0x2cb1569dbc9c9c64ac7c682acdf6515275277bd6"
var to1 = "0xafbfefa496ae205cf4e002dee11517e6d6da3ef6"
var from2 = "0x3ebe227e9fd42bb97b9a950e4a731d8975263812"
var tx1 = "0xc4690121c0a6cc6c0cb933b9551ae9926302a12a105ad8f24e50f8dadb4a6ece"
var tx2 = "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"
var addressIndexes = []*types.AddressIndex{
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  from1,
			Sequence: 1,
		},
		TxHash:        tx1,
		Value:         big.NewInt(-111),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
		CoupleAddress: to1,
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  to1,
			Sequence: 1,
		},
		TxHash:        tx1,
		Value:         big.NewInt(111),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
		CoupleAddress: from1,
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  from2,
			Sequence: 1,
		},
		TxHash:        tx2,
		Value:         big.NewInt(-222),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
		CoupleAddress: to1,
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  to1,
			Sequence: 2,
		},
		TxHash:        tx2,
		Value:         big.NewInt(222),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
		CoupleAddress: from2,
	},
}

var blockIndex = &types.BlockIndex{
	BlockNumber: "2018",
	Addresses: []types.AddressSequence{
		types.AddressSequence{Address: to1, Sequence: 2},
		types.AddressSequence{Address: from2, Sequence: 1},
		types.AddressSequence{Address: from1, Sequence: 1},
	},
}

func TestStoreAndMore(t *testing.T) {
	addressDB := memdb.New(comparer.DefaultComparer, 0)
	addressDAO := dao.NewMemDbDAO(addressDB)
	blockDB := memdb.New(comparer.DefaultComparer, 0)
	blockDAO := dao.NewMemDbDAO(blockDB)
	batchDB := memdb.New(comparer.DefaultComparer, 0)
	batchDAO := dao.NewMemDbDAO(batchDB)
	repo := NewLevelDBRepo(addressDAO, blockDAO, batchDAO)
	err := repo.Store(addressIndexes, blockIndex, true)
	assert.Nil(t, err)

	// GetTransactionByAddress
	total, addresses := repo.GetTransactionByAddress("wrong address", 10, 0, nil, nil)
	assert.Equal(t, 0, len(addresses))
	assert.Equal(t, 0, total)

	assertAddresses := func(asc bool) {
		// sort is desc, returned sequence is always 0 for now
		addressIndexes[3].AddressSequence.Sequence = 0
		addressIndexes[1].AddressSequence.Sequence = 0
		if asc {
			assert.True(t, reflect.DeepEqual(*addressIndexes[1], addresses[0]))
			assert.True(t, reflect.DeepEqual(*addressIndexes[3], addresses[1]))
		} else {
			assert.True(t, reflect.DeepEqual(*addressIndexes[3], addresses[0]))
			assert.True(t, reflect.DeepEqual(*addressIndexes[1], addresses[1]))
		}

	}

	total, addresses = repo.GetTransactionByAddress(to1, 10, 0, nil, nil)
	assert.Equal(t, 2, len(addresses))
	assertAddresses(false)
	assert.Equal(t, 2, total)
	// time range includes the transactions
	fromTime := big.NewInt(time.Now().Unix() - 100)
	toTime := big.NewInt(time.Now().Unix() + 100)
	total, addresses = repo.GetTransactionByAddress(to1, 10, 0, fromTime, toTime)
	assert.Equal(t, 2, len(addresses))
	assertAddresses(true)
	assert.Equal(t, 2, total)
	// edge case
	total, addresses = repo.GetTransactionByAddress(to1, 10, 0, blockTime, blockTime)
	assert.Equal(t, 2, len(addresses))
	assertAddresses(true)
	assert.Equal(t, 2, total)
	// time range does not include the transactions
	fromTime = big.NewInt(time.Now().Unix() + 1)
	toTime = big.NewInt(time.Now().Unix() + 100)
	total, addresses = repo.GetTransactionByAddress(to1, 10, 0, fromTime, toTime)
	assert.Equal(t, 0, len(addresses))
	assert.Equal(t, 0, total)

	// HandleReorg
	err = repo.HandleReorg(blockTime, blockIndex.Addresses)
	assert.Nil(t, err)
	total, addresses = repo.GetTransactionByAddress(to1, 10, 0, nil, nil)
	assert.Equal(t, 0, len(addresses))
	assert.Equal(t, 0, total)

	// GetLastNewHeadBlockInDB

	// GetFirstNewHeadBlockInDB

	// GetAllBatchStatuses

	// UpdateBatch

	// ReplaceBatch

	// GetBlocks
}
