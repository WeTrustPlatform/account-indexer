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
	"github.com/WeTrustPlatform/account-indexer/service"
	"github.com/WeTrustPlatform/account-indexer/watcher"
)

// Indexer fetch data from blockchain and store in a repository
type Indexer struct {
	IndexRepo repository.IndexRepo
	BatchRepo repository.BatchRepo
	bdChan    chan *types.BLockDetail
	watcher   watcher.Watcher
}

// NewIndexer create an Indexer
func NewIndexer(IndexRepo repository.IndexRepo, BatchRepo repository.BatchRepo, wa watcher.Watcher) Indexer {
	result := Indexer{IndexRepo: IndexRepo, BatchRepo: BatchRepo, watcher: wa}
	var sub service.IpcSubscriber
	sub = &result
	service.GetIpcManager().Subscribe(&sub)
	if wa == nil {
		result.watcher = watcher.NewNodeStatusWatcher(IndexRepo, BatchRepo)
	}
	return result
}

// IpcUpdated implements IpcSubscriber interface
func (indexer *Indexer) IpcUpdated(ipcPath string) {
	// finish any ongoing go-routines of this fetcher
	if indexer.bdChan != nil {
		// This should finish the realtimeIndex for loop
		close(indexer.bdChan)
	}
	time.Sleep(5 * time.Second)
	hasWatcher := false
	indexer.index(hasWatcher)
}

// IndexWithWatcher Entry point
func (indexer *Indexer) IndexWithWatcher() {
	indexer.index(true)
}

func (indexer *Indexer) index(hasWatcher bool) {
	fetcher, err := fetcher.NewChainFetch()
	if err != nil {
		log.Fatal("IPC path is not correct error:", err.Error())
	}
	latestBlock, err := fetcher.GetLatestBlock()
	if err != nil {
		log.Fatal("Can't get latest block, check IPC server. Error:", err)
		return
	}
	log.Println("IPC path is correct, latestBlock=" + latestBlock.String())
	batches := indexer.getBatches(latestBlock)
	mainWG := sync.WaitGroup{}
	if hasWatcher {
		mainWG.Add(2)
	} else {
		mainWG.Add(1)
	}

	batchWG := sync.WaitGroup{}
	batchWG.Add(len(batches))
	// index batches
	for _, bt := range batches {
		_bt := bt
		go func() {
			defer batchWG.Done()
			current := ""
			if _bt.Current != nil {
				current = _bt.Current.String()
			}
			indexer.batchIndex(_bt, ""+_bt.From.String()+"-"+_bt.To.String()+"-"+current+":")
		}()
	}
	// index realtime
	go func() {
		defer mainWG.Done()
		indexer.realtimeIndex(fetcher)
		log.Println("Realtime index is done")
	}()

	batchWG.Wait()
	log.Println("All batches are done")
	// All batches are done, start the watcher if needed
	if hasWatcher {
		go func() {
			defer mainWG.Done()
			indexer.watcher.Watch()
		}()
	}

	mainWG.Wait()
}

func (indexer *Indexer) getBatches(latestBlock *big.Int) []types.BatchStatus {
	allBatches := indexer.BatchRepo.GetAllBatchStatuses()
	batches := []types.BatchStatus{}
	now := big.NewInt(time.Now().Unix())
	if len(allBatches) == 0 {
		// Ethereum mainnet has genesis block as 0
		genesisBlock := big.NewInt(0)
		batches = GetInitBatches(common.GetConfig().NumBatch, genesisBlock, latestBlock)
	} else {
		// Get latest block in block database
		lastBlock, _ := indexer.IndexRepo.GetLastBlock()
		lastBlockNum := new(big.Int)
		lastBlockNum.SetString(lastBlock.BlockNumber, 10)
		allBatches := indexer.BatchRepo.GetAllBatchStatuses()
		found := false
		for _, batch := range allBatches {
			if !batch.IsDone() {
				if lastBlockNum != nil && lastBlockNum.Cmp(batch.From) == 0 {
					batch.To = latestBlock
					indexer.BatchRepo.ReplaceBatch(batch.From, latestBlock)
					found = true
					log.Println("Updated batch with from " + batch.From.String())
				}
				batches = append(batches, batch)
			}
		}
		if lastBlockNum != nil && !found {
			batch := types.BatchStatus{From: lastBlockNum, To: latestBlock, Step: byte(1), CreatedAt: now}
			batches = append(batches, batch)
		}
	}
	return batches
}

// RealtimeIndex newHead subscribe
func (indexer *Indexer) realtimeIndex(fetcher fetcher.Fetch) {
	indexer.bdChan = make(chan *types.BLockDetail)
	go fetcher.RealtimeFetch(indexer.bdChan)
	for {
		blockDetail, ok := <-indexer.bdChan
		if !ok {
			log.Println("Stopping realtimeIndex, ipc is switched?")
			break
		}
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
	fetcher, err := fetcher.NewChainFetch()
	if err != nil {
		log.Fatal("Can't connect to IPC server", err)
		return
	}
	for !batch.IsDone() {
		blockNumber := batch.Next()
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
		batch.UpdatedAt = big.NewInt(time.Now().Unix())
		err = indexer.BatchRepo.UpdateBatch(batch)
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
	return indexer.IndexRepo.Store(addressIndex, blockIndex, isBatch)
}

// CreateIndexData transforms blockchain data to our index data
func (indexer *Indexer) CreateIndexData(blockDetail *types.BLockDetail) ([]*types.AddressIndex, *types.BlockIndex) {
	addressIndex := make([]*types.AddressIndex, 0, 2*len(blockDetail.Transactions))
	blockIndex := &types.BlockIndex{
		BlockNumber: blockDetail.BlockNumber.String(),
		Addresses:   []types.AddressSequence{},
		Time:        blockDetail.Time,
		CreatedAt:   big.NewInt(time.Now().Unix()),
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

// GetInitBatches create batch initially
func GetInitBatches(numBatch int, genesisBlock *big.Int, latestBlock *big.Int) []types.BatchStatus {
	result := []types.BatchStatus{}
	now := big.NewInt(time.Now().Unix())
	for i := 0; i < numBatch; i++ {
		from := new(big.Int)
		from = from.Add(genesisBlock, big.NewInt(int64(i)))
		batch := types.BatchStatus{
			From:      from,
			To:        latestBlock,
			CreatedAt: now,
			Step:      byte(numBatch),
		}
		result = append(result, batch)
	}
	return result
}
