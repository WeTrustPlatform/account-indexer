package indexer

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/stretchr/testify/assert"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/dao"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
)

var blockTime = big.NewInt(time.Now().UnixNano())
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
		TxHash:        "0xtx1",
		Value:         big.NewInt(-111),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
		CoupleAddress: "to1",
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  "to1",
			Sequence: 1,
		},
		TxHash:        "0xtx1",
		Value:         big.NewInt(111),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
		CoupleAddress: "from1",
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  "from2",
			Sequence: 1,
		},
		TxHash:        "0xtx2",
		Value:         big.NewInt(-222),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
		CoupleAddress: "to1",
	},
	&types.AddressIndex{
		AddressSequence: types.AddressSequence{
			Address:  "to1",
			Sequence: 2,
		},
		TxHash:        "0xtx2",
		Value:         big.NewInt(222),
		Time:          blockTime,
		BlockNumber:   big.NewInt(2018),
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

	indexer := Indexer{}
	addressIndex, blockIndex := indexer.CreateIndexData(&blockDetail)
	if len(addressIndex) != len(expectedIndexes) {
		t.Error("Length of addressIndex is {}, expect {}", len(addressIndex), len(expectedIndexes))
	}
	for i, index := range addressIndex {
		if !reflect.DeepEqual(index, expectedIndexes[i]) {
			t.Error("Test failed at {}", i)
		}
	}
	if blockIndex.BlockNumber != "2018" {
		t.Error("BlockIndex - BlockNumber is not correct")
	}

	// blockIndex is not ordered due to map
	blockIndexAddresses := map[string]uint8{}
	for _, addressSequence := range blockIndex.Addresses {
		blockIndexAddresses[addressSequence.Address] = addressSequence.Sequence
	}

	if blockIndexAddresses["from1"] != uint8(1) || blockIndexAddresses["from2"] != uint8(1) || blockIndexAddresses["to1"] != uint8(2) {
		t.Error("BlockIndex - address sequences is not correct")
	}
}

func TestDivideRange(t *testing.T) {
	parentRange := Range{big.NewInt(2), big.NewInt(7)}
	range1, range2 := DivideRange(parentRange)
	if range1.From.String() != "2" || range1.To.String() != "4" {
		t.Error("Range1 is not correct")
	}
	if range2.From.String() != "5" || range2.To.String() != "7" {
		t.Error("Range2 is not correct")
	}
}

func TestGetBatches(t *testing.T) {
	blockDB := memdb.New(comparer.DefaultComparer, 0)
	blockDAO := dao.NewMemDbDAO(blockDB)
	batchDB := memdb.New(comparer.DefaultComparer, 0)
	batchDAO := dao.NewMemDbDAO(batchDB)
	repo := repository.NewLevelDBRepo(nil, blockDAO, batchDAO)
	indexer := Indexer{IpcPath: "", Repo: repo}
	// init data
	batch1 := types.BatchStatus{
		From:      big.NewInt(0),
		To:        big.NewInt(350),
		Current:   big.NewInt(200),
		UpdatedAt: big.NewInt(time.Now().Unix()),
	}
	batch2 := types.BatchStatus{
		From:      big.NewInt(351),
		To:        big.NewInt(700),
		Current:   big.NewInt(550),
		UpdatedAt: big.NewInt(time.Now().Unix()),
	}
	repo.UpdateBatch(batch1)
	repo.UpdateBatch(batch2)

	blockIndex := types.BlockIndex{
		BlockNumber: big.NewInt(800).String(),
		Addresses:   []types.AddressSequence{},
	}
	repo.SaveBlockIndex(&blockIndex)
	// Test: should add a new batch from latest block in DB to latest block in blockchain
	latestBlock := big.NewInt(900)
	batches := indexer.getBatches(latestBlock)
	assert.Equal(t, 3, len(batches), "Should add 1 batch")
	newBatch := batches[2]
	assert.Equal(t, big.NewInt(800), newBatch.From, "NewBatch From should be correct")
	assert.Equal(t, latestBlock, newBatch.To, "NewBatch To should be correct")

	// Init data next, assuming batch stop at 850
	current := big.NewInt(850)
	newBatch.Current = current
	newBatch.UpdatedAt = big.NewInt(time.Now().Unix())
	repo.UpdateBatch(newBatch)

	// Test: should not add a new batch, reuse the last batch with updated "To"
	latestBlock = big.NewInt(1000)
	batches = indexer.getBatches(latestBlock)
	assert.Equal(t, 3, len(batches), "Should not add 1 batch")
	newBatch = batches[2]
	assert.Equal(t, big.NewInt(800), newBatch.From, "NewBatch From should be correct")
	assert.Equal(t, latestBlock, newBatch.To, "NewBatch To should be correct")
	assert.Equal(t, current, newBatch.Current, "NewBatch Current should be correct")
}
