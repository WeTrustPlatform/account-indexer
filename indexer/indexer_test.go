package indexer

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
)

var blockTime = *big.NewInt(time.Now().UnixNano())
var blockDetail = types.BLockDetail{
	BlockNumber: *big.NewInt(2018),
	Time:        blockTime,
	Transactions: []types.TransactionDetail{
		types.TransactionDetail{
			From:   "from1",
			To:     "to1",
			TxHash: "0xtx1",
			Value:  *big.NewInt(111),
		},
		types.TransactionDetail{
			From:   "from2",
			To:     "to2",
			TxHash: "0xtx2",
			Value:  *big.NewInt(222),
		},
	},
}

var expectedIndexes = []types.AddressIndex{
	types.AddressIndex{
		Address:     "from1",
		TxHash:      "0xtx1",
		Value:       *big.NewInt(-111),
		Time:        blockTime,
		BlockNumber: *big.NewInt(2018),
	},
	types.AddressIndex{
		Address:     "to1",
		TxHash:      "0xtx1",
		Value:       *big.NewInt(111),
		Time:        blockTime,
		BlockNumber: *big.NewInt(2018),
	},
	types.AddressIndex{
		Address:     "from2",
		TxHash:      "0xtx2",
		Value:       *big.NewInt(-222),
		Time:        blockTime,
		BlockNumber: *big.NewInt(2018),
	},
	types.AddressIndex{
		Address:     "to2",
		TxHash:      "0xtx2",
		Value:       *big.NewInt(222),
		Time:        blockTime,
		BlockNumber: *big.NewInt(2018),
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
	addressIndex, blockIndex := indexer.CreateIndexData(blockDetail)
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
	if !reflect.DeepEqual(blockIndex.Addresses, []string{"from1", "to1", "from2", "to2"}) {
		t.Error("BlockIndex - Addresses is not correct")
	}
}

func TestDivideRange(t *testing.T) {
	parentRange := Range{big.NewInt(2), big.NewInt(7)}
	range1, range2 := DivideRange(parentRange)
	if range1.From.String() != "2" || range1.To.String() != "4" {
		t.Error("Range1 is not correct")
	}
	if range2.From.String() != "4" || range2.To.String() != "8" {
		t.Error("Range2 is not correct")
	}
}
