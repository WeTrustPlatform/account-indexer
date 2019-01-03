package indexer

import (
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/fetcher"
	"github.com/WeTrustPlatform/account-indexer/repository"
)

// Indexer fetch data from blockchain and store in a repository
type Indexer struct {
	// Fetcher Fetch
	IpcPath string
	Repo    repository.Repository
}

// Range from block - to block
type Range struct {
	// Inclusive
	From *big.Int
	// Exclusive
	To *big.Int
}

// DivideRange performance states that having > 2 goroutines have same performance
// so let's go with 2 goroutines
// To and From are inclusive
func DivideRange(parent Range) (Range, Range) {
	minusFrom := new(big.Int)
	minusFrom = minusFrom.Neg(parent.From)
	distance := new(big.Int)
	distance = distance.Add(parent.To, minusFrom)
	distance = distance.Div(distance, big.NewInt(2))
	middle := new(big.Int)
	middle = middle.Add(parent.From, distance)
	middlePlus1 := new(big.Int)
	middlePlus1 = middlePlus1.Add(middle, big.NewInt(1))
	to := new(big.Int)
	to = to.Set(parent.To)
	range1From := new(big.Int)
	range1From = range1From.Set(parent.From)
	range1 := Range{From: range1From, To: middle}
	range2 := Range{From: middlePlus1, To: to}
	return range1, range2
}

// Index Entry point
func (indexer *Indexer) Index() {
	fetcher, err := fetcher.NewChainFetch(indexer.IpcPath)
	allBatches := indexer.Repo.GetAllBatchStatuses()
	latestBlock, err := fetcher.GetLatestBlock()
	if err != nil {
		log.Fatal("Can't get latest block, check IPC server", err)
		return
	}
	if len(allBatches) == 0 {
		// index from genesis
		indexer.IndexFromGenesis(latestBlock)
	} else {
		allBatches := indexer.Repo.GetAllBatchStatuses()
		for _, batch := range allBatches {
			if batch.To.Cmp(batch.Current) > 0 {
				go indexer.indexByRange(Range{From: batch.Current, To: batch.To}, "From "+batch.Current.String()+";To "+batch.To.String())
			}
		}
		// Get latest block in block database
		lastNewHeadBlockInDB := indexer.Repo.GetLastNewHeadBlockInDB()
		if lastNewHeadBlockInDB != nil {
			go indexer.indexByRange(Range{From: lastNewHeadBlockInDB, To: latestBlock}, "From "+lastNewHeadBlockInDB.String()+" to latest block")
		}
	}

	go indexer.RealtimeIndex(fetcher)
}

// RealtimeIndex newHead subscribe
func (indexer *Indexer) RealtimeIndex(fetcher fetcher.Fetch) {
	indexerChannel := make(chan types.BLockDetail)
	// go indexer.Fetcher.RealtimeFetch(indexerChannel)
	go fetcher.RealtimeFetch(indexerChannel)
	for {
		blockDetail := <-indexerChannel
		fmt.Println("indexer: Received BlockDetail " + blockDetail.BlockNumber.String())
		isBatch := false
		indexer.processBlock(blockDetail, isBatch)
	}
}

// IndexFromGenesis index from block 1
func (indexer *Indexer) IndexFromGenesis(latestBlock *big.Int) {
	start := time.Now()
	// TODO: change 1 to genesis block
	range1, range2 := DivideRange(Range{big.NewInt(1), latestBlock})
	fmt.Println(range1)
	fmt.Println(range2)
	for i := 0; i < 5; i++ {
		fmt.Println(5 - i)
		time.Sleep(time.Second)
	}
	// https://nathanleclaire.com/blog/2014/02/15/how-to-wait-for-all-goroutines-to-finish-executing-before-continuing/
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		indexer.indexByRange(range1, "1")
	}()
	go func() {
		defer wg.Done()
		indexer.indexByRange(range2, "2")
	}()

	wg.Wait()
	duration := time.Since(start)
	s := fmt.Sprintf("%f", duration.Minutes())
	fmt.Println("Index " + latestBlock.String() + " block took " + s + " minutes")
}

// from: inclusive, to: exclusive
func (indexer *Indexer) indexByRange(rg Range, tag string) {
	start := time.Now()
	from := rg.From
	to := rg.To
	fetcher, err := fetcher.NewChainFetch(indexer.IpcPath)
	if err != nil {
		log.Fatal("Can't connect to IPC server", err)
		return
	}
	blockNumber := new(big.Int)
	for blockNumber.Set(from); blockNumber.Cmp(to) <= 0; blockNumber = blockNumber.Add(blockNumber, big.NewInt(int64(1))) {
		// fmt.Println("indexer: Received BlockDetail " + blockNumber.String())
		blockDetail, err := fetcher.FetchABlock(blockNumber)
		if err == nil {
			fmt.Println(tag + " indexer: Received BlockDetail " + blockDetail.BlockNumber.String())
			isBatch := true
			indexer.processBlock(blockDetail, isBatch)
			batchStatus := types.BatchStatus{
				From:      rg.From,
				To:        rg.To,
				Current:   blockNumber,
				UpdatedAt: big.NewInt(time.Now().UnixNano()),
			}
			indexer.Repo.UpdateBatch(batchStatus)
		} else {
			fmt.Println(tag + " indexer: cannot get block " + blockNumber.String() + " , error is " + err.Error())
			// TODO: log warning
		}
	}
	duration := time.Since(start)
	s := fmt.Sprintf("%f", duration.Minutes())
	fmt.Println(tag + " is done in " + s + " minutes")
}

func (indexer *Indexer) processBlock(blockDetail types.BLockDetail, isBatch bool) {
	addressIndex, blockIndex := indexer.CreateIndexData(blockDetail)
	indexer.Repo.Store(addressIndex, blockIndex, isBatch)
	fmt.Println("indexer: Saved block " + blockDetail.BlockNumber.String() + " to Repository already")
}

// CreateIndexData transforms blockchain data to our index data
func (indexer *Indexer) CreateIndexData(blockDetail types.BLockDetail) ([]types.AddressIndex, types.BlockIndex) {
	addressIndex := make([]types.AddressIndex, 0, 2*len(blockDetail.Transactions))
	blockIndex := types.BlockIndex{
		BlockNumber: blockDetail.BlockNumber.String(),
		Addresses:   []types.AddressSequence{},
	}
	sequenceMap := map[string]uint8{}

	for _, transaction := range blockDetail.Transactions {
		// TODO: resolve pointer issue
		posValue := transaction.Value
		negValue := transaction.Value.Mul(&posValue, big.NewInt(-1))

		fromIndex := types.AddressIndex{
			TxHash:        transaction.TxHash,
			Value:         *negValue,
			Time:          blockDetail.Time,
			BlockNumber:   blockDetail.BlockNumber,
			CoupleAddress: transaction.To,
		}

		toIndex := types.AddressIndex{
			TxHash:        transaction.TxHash,
			Value:         posValue,
			Time:          blockDetail.Time,
			BlockNumber:   blockDetail.BlockNumber,
			CoupleAddress: transaction.From,
		}
		if _, ok := sequenceMap[transaction.From]; !ok {
			sequenceMap[transaction.From] = 0
			// blockIndex.Addresses = append(blockIndex.Addresses, transaction.From)
		}
		sequenceMap[transaction.From]++

		if _, ok := sequenceMap[transaction.To]; !ok {
			sequenceMap[transaction.To] = 0
			// blockIndex.Addresses = append(blockIndex.Addresses, transaction.To)
		}
		sequenceMap[transaction.To]++

		fromIndex.Address = transaction.From
		fromIndex.Sequence = sequenceMap[transaction.From]
		toIndex.Address = transaction.To
		toIndex.Sequence = sequenceMap[transaction.To]
		addressIndex = append(addressIndex, fromIndex, toIndex)
	}
	for k, v := range sequenceMap {
		blockIndex.Addresses = append(blockIndex.Addresses, types.AddressSequence{Address: k, Sequence: v})
	}
	return addressIndex, blockIndex
}
