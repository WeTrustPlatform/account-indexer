package indexer

import (
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
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

// Index Entry point
func (indexer *Indexer) Index() {
	fetcher, err := fetcher.NewChainFetch(indexer.IpcPath)
	if err != nil {
		log.Fatal("IPC path is not correct")
	}
	latestBlock, err := fetcher.GetLatestBlock()
	if err != nil {
		log.Fatal("Can't get latest block, check IPC server. Error:", err)
		return
	}
	log.Println("IPC path is correct, latestBlock=" + latestBlock.String())
	batches := indexer.getBatches(latestBlock)
	wg := sync.WaitGroup{}
	wg.Add(len(batches) + 1)
	// index batches
	for _, bt := range batches {
		_bt := bt
		go func() {
			defer wg.Done()
			current := ""
			if _bt.Current != nil {
				current = _bt.Current.String()
			}
			indexer.batchIndex(_bt, ""+_bt.From.String()+"-"+_bt.To.String()+"-"+current+":")
		}()
	}
	// index realtime
	go func() {
		defer wg.Done()
		indexer.RealtimeIndex(fetcher)
	}()

	wg.Wait()
}

func (indexer *Indexer) getBatches(latestBlock *big.Int) []types.BatchStatus {
	allBatches := indexer.Repo.GetAllBatchStatuses()
	batches := []types.BatchStatus{}
	if len(allBatches) == 0 {
		// Ethereum mainnet has genesis block as 0
		genesisBlock := big.NewInt(0)
		range1, range2 := DivideRange(Range{genesisBlock, latestBlock})
		batches = append(batches, types.BatchStatus{From: range1.From, To: range1.To}, types.BatchStatus{From: range2.From, To: range2.To})
	} else {
		// Get latest block in block database
		lastNewHeadBlockInDB := indexer.Repo.GetLastNewHeadBlockInDB()
		allBatches := indexer.Repo.GetAllBatchStatuses()
		found := false
		for _, batch := range allBatches {
			if batch.To.Cmp(batch.Current) > 0 {
				if lastNewHeadBlockInDB != nil && lastNewHeadBlockInDB.Cmp(batch.From) == 0 {
					batch.To = latestBlock
					indexer.Repo.ReplaceBatch(batch.From, latestBlock)
					found = true
					log.Println("Updated batch with from " + batch.From.String())
				}
				batches = append(batches, batch)
			}
		}
		if lastNewHeadBlockInDB != nil && !found {
			batch := types.BatchStatus{From: lastNewHeadBlockInDB, To: latestBlock}
			batches = append(batches, batch)
		}
	}
	return batches
}

// RealtimeIndex newHead subscribe
func (indexer *Indexer) RealtimeIndex(fetcher fetcher.Fetch) {
	indexerChannel := make(chan *types.BLockDetail)
	// go indexer.Fetcher.RealtimeFetch(indexerChannel)
	go fetcher.RealtimeFetch(indexerChannel)
	for {
		blockDetail := <-indexerChannel
		log.Printf("indexer: Received BlockDetail %v blockTime: %v\n", blockDetail.BlockNumber.String(), common.UnmarshallIntToTime(blockDetail.Time))
		isBatch := false
		indexer.processBlock(blockDetail, isBatch)
	}
}

// from: inclusive, to: exclusive
func (indexer *Indexer) batchIndex(batch types.BatchStatus, tag string) {
	log.Println("indexByRange, tag=" + tag)
	for i := 0; i < 5; i++ {
		log.Printf("Tag: %v, time to start: %v seconds \n", tag, (5 - i))
		time.Sleep(time.Second)
	}
	start := time.Now()
	from := batch.Current
	if from == nil {
		from = batch.From
	}
	to := batch.To
	fetcher, err := fetcher.NewChainFetch(indexer.IpcPath)
	if err != nil {
		log.Fatal("Can't connect to IPC server", err)
		return
	}
	blockNumber := new(big.Int)
	for blockNumber.Set(from); blockNumber.Cmp(to) <= 0; blockNumber = blockNumber.Add(blockNumber, big.NewInt(int64(1))) {
		blockDetail, err := fetcher.FetchABlock(blockNumber)
		if err != nil {
			log.Fatal(tag + " indexer: cannot get block " + blockNumber.String() + " , error is " + err.Error())
		}
		// log.Println(tag + " indexer: Received BlockDetail " + blockDetail.BlockNumber.String())
		isBatch := true
		err = indexer.processBlock(blockDetail, isBatch)
		if err != nil {
			log.Fatal(tag + " indexer: cannot process block " + blockNumber.String() + " , error is " + err.Error())
		}
		batchStatus := types.BatchStatus{
			From:      batch.From,
			To:        batch.To,
			Current:   blockNumber,
			UpdatedAt: big.NewInt(time.Now().Unix()),
		}
		err = indexer.Repo.UpdateBatch(batchStatus)
		if err != nil {
			log.Fatal(tag + " indexer: cannot update batch for process block " + blockNumber.String() + " , error is " + err.Error())
		}
	}
	duration := time.Since(start)
	s := fmt.Sprintf("%f", duration.Minutes())
	log.Println(tag + " is done in " + s + " minutes")
}

func (indexer *Indexer) processBlock(blockDetail *types.BLockDetail, isBatch bool) error {
	addressIndex, blockIndex := indexer.CreateIndexData(blockDetail)
	return indexer.Repo.Store(addressIndex, blockIndex, isBatch)
}

// CreateIndexData transforms blockchain data to our index data
func (indexer *Indexer) CreateIndexData(blockDetail *types.BLockDetail) ([]*types.AddressIndex, *types.BlockIndex) {
	addressIndex := make([]*types.AddressIndex, 0, 2*len(blockDetail.Transactions))
	blockIndex := &types.BlockIndex{
		BlockNumber: blockDetail.BlockNumber.String(),
		Addresses:   []types.AddressSequence{},
		Time:        blockDetail.Time,
	}
	sequenceMap := map[string]uint8{}

	for _, transaction := range blockDetail.Transactions {
		posValue := transaction.Value
		negValue := new(big.Int)
		negValue = negValue.Mul(posValue, big.NewInt(-1))
		to := transaction.To
		isNilTo := false
		if to == "" {
			to = common.AddressZero
			isNilTo = true
		}

		fromIndex := types.AddressIndex{
			TxHash:        transaction.TxHash,
			Value:         negValue,
			Time:          blockDetail.Time,
			BlockNumber:   blockDetail.BlockNumber,
			CoupleAddress: to,
		}
		if _, ok := sequenceMap[transaction.From]; !ok {
			sequenceMap[transaction.From] = 0
		}
		sequenceMap[transaction.From]++
		fromIndex.Address = transaction.From
		fromIndex.Sequence = sequenceMap[transaction.From]
		addressIndex = append(addressIndex, &fromIndex)

		if !isNilTo {
			toIndex := types.AddressIndex{
				TxHash:        transaction.TxHash,
				Value:         posValue,
				Time:          blockDetail.Time,
				BlockNumber:   blockDetail.BlockNumber,
				CoupleAddress: transaction.From,
			}
			if _, ok := sequenceMap[transaction.To]; !ok {
				sequenceMap[transaction.To] = 0
			}
			sequenceMap[transaction.To]++
			toIndex.Address = transaction.To
			toIndex.Sequence = sequenceMap[transaction.To]
			addressIndex = append(addressIndex, &toIndex)
		}
	}
	for k, v := range sequenceMap {
		blockIndex.Addresses = append(blockIndex.Addresses, types.AddressSequence{Address: k, Sequence: v})
	}
	return addressIndex, blockIndex
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
