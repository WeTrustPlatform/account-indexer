// +build int

package int

import (
	"math/big"
	"strings"
	"testing"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/fetcher"
	"github.com/WeTrustPlatform/account-indexer/indexer"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue/dao"
	"github.com/WeTrustPlatform/account-indexer/service"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
)

const (
	IPC = "wss://mainnet.kivutar.me:8546/2KT179di"
	// IPC = "wss://mainnet.infura.io/_ws"
)

func TestContractCreation(t *testing.T) {
	// Setup
	ipcs := []string{IPC}
	t.Logf("TestContractCreation ipcs=%v \n ", ipcs)
	service.GetIpcManager().SetIPC(ipcs)
	fetcher, err := fetcher.NewChainFetch()
	assert.Nil(t, err)
	blockNumber := big.NewInt(6808718)
	// Run Test
	blockDetail, err := fetcher.FetchABlock(blockNumber.Int64())
	assert.Nil(t, err)
	// log.Println(blockDetail)
	idx := NewTestIndexer()
	isBatch := true
	idx.ProcessBlock(blockDetail, isBatch)
	// Confirm contract created tx
	contract := "0x4a6ead96974679957a17d2f9c7835a3da7ddf91d"
	fromTime, _ := common.StrToTime("2018-12-01T00:00:00")
	toTime, _ := common.StrToTime("2018-12-01T23:59:59")
	total, addressIndexes := idx.IndexRepo.GetTransactionByAddress(contract, 10, 0, fromTime, toTime)
	assert.Equal(t, 1, total)
	tx := addressIndexes[0].TxHash
	assert.True(t, strings.EqualFold("0x61278dd960415eadf11cfe17a6c38397af658e77bbdd367db70e19ee3a193bdd", tx))
	tm := common.UnmarshallIntToTime(addressIndexes[0].Time)
	t.Logf("TestContractCreation found transaction at %v \n", tm)
}

func TestFailedTransaction(t *testing.T) {
	// If a transaction is failed, do not index it
	// Setup
	ipcs := []string{IPC}
	t.Logf("TestFailedTransaction ipcs=%v \n ", ipcs)
	service.GetIpcManager().SetIPC(ipcs)
	fetcher, err := fetcher.NewChainFetch()
	assert.Nil(t, err)
	blockNumber := big.NewInt(7156456)
	// This block has 162 transactions but 3 failed
	blockDetail, err := fetcher.FetchABlock(blockNumber.Int64())
	assert.Nil(t, err)
	numSuccess := 0
	for _, tx := range blockDetail.Transactions {
		if tx.Status {
			numSuccess++
		}
	}

	// Test fetcher
	assert.Equal(t, 159, numSuccess)
	found := false
	for _, tx := range blockDetail.Transactions {
		// one of the failed transaction, but it's still included in the block
		if tx.TxHash == "0x62a1c5a48137c5a649b808b6756a9d4d2fd500a7bde984fe671c95ad911639d5" {
			assert.False(t, tx.Status)
			found = true
			break
		}
	}
	assert.True(t, found)

	// Test indexer
	idx := NewTestIndexer()
	isBatch := true
	idx.ProcessBlock(blockDetail, isBatch)
	address := "0x38F88D57A2589C4972eA360e3FC38E92bb7dd110"
	fromTime, _ := common.StrToTime("2019-02-01T00:00:00")
	toTime, _ := common.StrToTime("2019-02-01T23:59:59")
	total, addressIndexes := idx.IndexRepo.GetTransactionByAddress(address, 10, 0, fromTime, toTime)
	assert.Equal(t, 1, total)
	assert.False(t, addressIndexes[0].Status)
}

func NewTestIndexer() indexer.Indexer {
	addressDB := memdb.New(comparer.DefaultComparer, 0)
	addressDAO := dao.NewMemDbDAO(addressDB)
	blockDB := memdb.New(comparer.DefaultComparer, 0)
	blockDAO := dao.NewMemDbDAO(blockDB)
	batchDB := memdb.New(comparer.DefaultComparer, 0)
	batchDAO := dao.NewMemDbDAO(batchDB)
	indexRepo := keyvalue.NewKVIndexRepo(addressDAO, blockDAO)
	batchRepo := keyvalue.NewKVBatchRepo(batchDAO)
	idx := indexer.NewIndexer(indexRepo, batchRepo, nil)
	return idx
}
