package indexer

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue"
	"github.com/stretchr/testify/assert"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue/dao"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
)

var blockTime = big.NewInt(time.Now().Unix())
var blockDetail = types.BLockDetail{
	BlockNumber: big.NewInt(2018),
	Time:        blockTime,
	Transactions: []types.TransactionDetail{
		types.TransactionDetail{
			From:   "from1",
			To:     "to1",
			TxHash: "0xtx1",
			Value:  big.NewInt(111),
		},
		types.TransactionDetail{
			From:   "from2",
			To:     "to1", // This demonstrates 2 transactions with same "to" address, Sequence should be increased
			TxHash: "0xtx2",
			Value:  big.NewInt(222),
		},
	},
}

var expectedIndexes = []*types.AddressIndex{
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  "from1",
			Sequence: 1,
		},
		TxHash: "0xtx1",
		Value:  big.NewInt(-111),
		Time:   blockTime,
		// BlockNumber:   big.NewInt(2018),
		CoupleAddress: "to1",
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  "to1",
			Sequence: 1,
		},
		TxHash: "0xtx1",
		Value:  big.NewInt(111),
		Time:   blockTime,
		// BlockNumber:   big.NewInt(2018),
		CoupleAddress: "from1",
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  "from2",
			Sequence: 1,
		},
		TxHash: "0xtx2",
		Value:  big.NewInt(-222),
		Time:   blockTime,
		// BlockNumber:   big.NewInt(2018),
		CoupleAddress: "to1",
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  "to1",
			Sequence: 2,
		},
		TxHash: "0xtx2",
		Value:  big.NewInt(222),
		Time:   blockTime,
		// BlockNumber:   big.NewInt(2018),
		CoupleAddress: "from2",
	},
}

type MockFetch struct{}

func (mf MockFetch) RealtimeFetch(ch chan<- types.BLockDetail) {
	ch <- blockDetail
}
func (mf MockFetch) FetchABlock(blockNumber *big.Int) (types.BLockDetail, error) {
	return types.BLockDetail{}, nil
}

type MockRepo struct {
	addressIndex []types.AddressIndex
	blockIndex   types.BlockIndex
}

func (mr MockRepo) Store(indexData []types.AddressIndex, blockIndex types.BlockIndex) {
	mr.addressIndex = indexData
	mr.blockIndex = blockIndex
}

func (mr MockRepo) HandleReorg(blockIndex string, reorgAddresses []string) {
}

func (mr MockRepo) Get(address string) []types.AddressIndex {
	return nil
}

type MockWatcher struct {
	watched bool
}

func (mw MockWatcher) Watch() {
	mw.watched = true
}

// TODO: fix me
// func TestIndex(t *testing.T) {
// 	mockRepo := MockRepo{}
// 	indexer := Indexer{
// 		// Fetcher: MockFetch{},
// 		IpcPath: "",
// 		Repo:    mockRepo,
// 	}
// 	time.Sleep(time.Second * 1)
// 	indexData := mockRepo.addressIndex
// 	for i, index := range indexData {
// 		if !reflect.DeepEqual(index, expectedIndexes[i]) {
// 			t.Error("Test failed at {}", i)
// 		}
// 	}
// }

func TestCreateIndexData(t *testing.T) {

	idx := Indexer{}
	addressIndex, blockIndex := idx.CreateIndexData(&blockDetail)
	assert.Equal(t, len(expectedIndexes), len(addressIndex))
	for i, index := range addressIndex {
		assert.True(t, reflect.DeepEqual(index, expectedIndexes[i]))
	}
	assert.Equal(t, "2018", blockIndex.BlockNumber)
	assert.Equal(t, blockTime, blockIndex.Time)

	// blockIndex is not ordered due to map
	blockIndexAddresses := map[string]uint8{}
	for _, addressSequence := range blockIndex.Addresses {
		blockIndexAddresses[addressSequence.Address] = addressSequence.Sequence
	}
	assert.Equal(t, uint8(1), blockIndexAddresses["from1"])
	assert.Equal(t, uint8(1), blockIndexAddresses["from2"])
	assert.Equal(t, uint8(2), blockIndexAddresses["to1"])
}

func TestGetInitBatches(t *testing.T) {
	genesisBlock := big.NewInt(0)
	latestBlock := big.NewInt(10)
	numBatch := 3
	batches := GetInitBatches(numBatch, genesisBlock, latestBlock)
	assert.Equal(t, 3, len(batches))
	assert.Equal(t, big.NewInt(0), batches[0].From)
	assert.Equal(t, big.NewInt(1), batches[1].From)
	assert.Equal(t, big.NewInt(2), batches[2].From)
	assert.Equal(t, byte(3), batches[0].Step)
	assert.Equal(t, byte(3), batches[1].Step)
	assert.Equal(t, byte(3), batches[2].Step)
}

func TestGetBatches(t *testing.T) {
	blockDB := memdb.New(comparer.DefaultComparer, 0)
	blockDAO := dao.NewMemDbDAO(blockDB)
	batchDB := memdb.New(comparer.DefaultComparer, 0)
	batchDAO := dao.NewMemDbDAO(batchDB)
	indexRepo := keyvalue.NewKVIndexRepo(nil, blockDAO)
	batchRepo := keyvalue.NewKVBatchRepo(batchDAO)
	idx := NewIndexer(indexRepo, batchRepo, nil)
	// init data
	batch1 := types.BatchStatus{
		From:      big.NewInt(0),
		To:        big.NewInt(700),
		Step:      byte(2),
		Current:   big.NewInt(200),
		CreatedAt: big.NewInt(time.Now().Unix() - 1000),
		UpdatedAt: big.NewInt(time.Now().Unix()),
	}
	batch2 := types.BatchStatus{
		From:      big.NewInt(1),
		To:        big.NewInt(700),
		Step:      byte(2),
		Current:   big.NewInt(231),
		CreatedAt: big.NewInt(time.Now().Unix() - 1000),
		UpdatedAt: big.NewInt(time.Now().Unix()),
	}
	batchRepo.UpdateBatch(batch1)
	batchRepo.UpdateBatch(batch2)

	blockIndex := types.BlockIndex{
		BlockNumber: big.NewInt(800).String(),
		Addresses:   []types.AddressSequence{},
		Time:        big.NewInt(time.Now().Unix()),
		CreatedAt:   big.NewInt(time.Now().Unix()),
	}
	indexRepo.SaveBlockIndex(&blockIndex)
	// Test: should add a new batch from latest block in DB to latest block in blockchain
	latestBlock := big.NewInt(900)
	batches := idx.getBatches(latestBlock)
	assert.Equal(t, 3, len(batches), "Should add 1 batch")
	newBatch := batches[2]
	assert.Equal(t, big.NewInt(800), newBatch.From)
	assert.Equal(t, latestBlock, newBatch.To)
	assert.Equal(t, byte(1), newBatch.Step)

	// Init data next, assuming batch stop at 850
	current := big.NewInt(850)
	newBatch.Current = current
	newBatch.UpdatedAt = big.NewInt(time.Now().Unix())
	batchRepo.UpdateBatch(newBatch)

	// Test: should not add a new batch, reuse the last batch with updated "To"
	latestBlock = big.NewInt(1000)
	batches = idx.getBatches(latestBlock)
	assert.Equal(t, 3, len(batches), "Should not add 1 batch")
	newBatch = batches[2]
	assert.Equal(t, big.NewInt(800), newBatch.From, "NewBatch From should be correct")
	assert.Equal(t, latestBlock, newBatch.To, "NewBatch To should be correct")
	assert.Equal(t, current, newBatch.Current, "NewBatch Current should be correct")
}

func TestWatchAfterBatch(t *testing.T) {
	// TODO
}
