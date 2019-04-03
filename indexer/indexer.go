package indexer

import (
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/common/config"
	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/fetcher"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/WeTrustPlatform/account-indexer/service"
	"github.com/WeTrustPlatform/account-indexer/watcher"
)

// Indexer fetch data from blockchain and store in a repository
type Indexer struct {
	IndexRepo       repository.IndexRepo
	BatchRepo       repository.BatchRepo
	bdChan          chan *types.BLockDetail
	watcher         watcher.Watcher
	realtimeFetcher *fetcher.ChainFetch
	stopChan        chan struct{}
}

// NewIndexer create an Indexer
func NewIndexer(IndexRepo repository.IndexRepo, BatchRepo repository.BatchRepo, wa watcher.Watcher) Indexer {
	result := Indexer{IndexRepo: IndexRepo, BatchRepo: BatchRepo, watcher: wa}
	if wa == nil {
		wt := watcher.NewNodeStatusWatcher(IndexRepo, BatchRepo)
		result.watcher = &wt
	}
	return result
}

// IpcUpdated implements IpcSubscriber interface
func (indexer *Indexer) IpcUpdated(ipcPath string) {
	if indexer.realtimeFetcher != nil {
		// let the old realtime fetch go, no need to give it new ipc
		indexer.realtimeFetcher.IpcUpdated()
	}
	log.Println("Indexer: stopping all batch index goroutines, waiting for 30s")
	// finish any ongoing go-routines of this fetcher
	if indexer.stopChan != nil {
		close(indexer.stopChan)
	}
	time.Sleep(30 * time.Second)
	log.Println("Indexer: stopped all batch index goroutines")
	indexAfterIPCChange := true
	indexer.index(indexAfterIPCChange)
}

// Name implements IpcSubscriber interface
func (indexer *Indexer) Name() string {
	return "Indexer"
}

// FirstIndex Entry point
func (indexer *Indexer) FirstIndex() {
	indexAfterIPCChange := false
	indexer.index(indexAfterIPCChange)
}

func (indexer *Indexer) index(indexAfterIPCChange bool) {
	service.GetIpcManager().EnableSwitchIPC()
	indexer.stopChan = make(chan struct{})
	if !indexAfterIPCChange {
		// don't subscribe again
		var sub service.IpcSubscriber = indexer
		service.GetIpcManager().Subscribe(&sub)
	}

	indexer.createRealtimeFetcher()
	if indexer.realtimeFetcher == nil {
		log.Println("Indexer: index stopped because cannot create new fetch for realtime goroutine")
		return
	}

	latestBlock, err := indexer.realtimeFetcher.GetLatestBlock()
	if err != nil {
		log.Printf("Indexer: Can't get latest block, check IPC server. Error: %v \n", err.Error())
		return
	}
	log.Printf("Indexer: IPC path %v is correct, latestBlock=%v \n", service.GetIpcManager().GetIPC(), latestBlock.String())
	batches := indexer.getBatches(latestBlock)
	mainWG := sync.WaitGroup{}
	mainWG.Add(2)
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
			tag := "" + _bt.From.String() + "-" + _bt.To.String() + "-" + current + ":"
			indexer.batchIndex(_bt, indexer.stopChan, tag)
		}()
	}
	// index realtime
	go func() {
		defer mainWG.Done()
		indexer.realtimeIndex()
	}()

	batchWG.Wait()
	log.Println("Indexer: All batches are done, starting watcher")
	go func() {
		defer mainWG.Done()
		indexer.watcher.Watch()
	}()

	mainWG.Wait()
}

func (indexer *Indexer) getBatches(latestBlock *big.Int) []types.BatchStatus {
	allBatches := indexer.BatchRepo.GetAllBatchStatuses()
	batches := []types.BatchStatus{}
	now := big.NewInt(time.Now().Unix())
	if len(allBatches) == 0 {
		// Ethereum mainnet has genesis block as 0
		genesisBlock := big.NewInt(0)
		batches = GetInitBatches(config.GetConfig().NumBatch, genesisBlock, latestBlock)
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
					log.Println("Indexer: Updated batch with from " + batch.From.String())
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
func (indexer *Indexer) realtimeIndex() {
	log.Println("Indexer: Starting realtime index")
	indexer.bdChan = make(chan *types.BLockDetail)
	go indexer.realtimeFetcher.RealtimeFetch(indexer.bdChan)
	for {
		blockDetail, ok := <-indexer.bdChan
		if !ok {
			log.Println("Indexer: Stopping realtimeIndex, ipc is switched?")
			break
		}
		log.Printf("Indexer: realtimeIndex - received BlockDetail %v blockTime: %v\n", blockDetail.BlockNumber.String(), common.UnmarshallIntToTime(blockDetail.Time))
		go func(blockDetail *types.BLockDetail) {
			indexer.processRealtimeBlock(blockDetail)
		}(blockDetail)
	}
	indexer.realtimeFetcher = nil
	log.Println("Indexer: Stopped realtimeIndex")
}

// from: inclusive, to: exclusive
func (indexer *Indexer) batchIndex(batch types.BatchStatus, stop chan struct{}, tag string) {
	log.Println("Indexer: start batchIndex, tag=" + tag)
	start := time.Now()
	fetcher, err := fetcher.NewChainFetch()
	if err != nil {
		log.Println(tag+"Indexer: batchIndex can't connect to IPC server", err)
		return
	}
	i := 0
	for !batch.IsDone() {
		blockNumber := batch.Next()
		isRecpRelax := false
		blockDetail, err := fetcher.FetchABlock(blockNumber.Int64(), isRecpRelax)
		if err != nil {
			// Finish the go-routines, someone will restart index()
			log.Println(tag + " Indexer: cannot get block " + blockNumber.String() + " , error is " + err.Error())
			break
		}
		isBatch := true
		err = indexer.ProcessBlock(blockDetail, isBatch)
		if err != nil {
			log.Fatal(tag + " Indexer: cannot process block " + blockNumber.String() + " , error is " + err.Error())
		}
		batch.UpdatedAt = big.NewInt(time.Now().Unix())
		err = indexer.BatchRepo.UpdateBatch(batch)
		if err != nil {
			log.Fatal(tag + " Indexer: cannot update batch for process block " + blockNumber.String() + " , error is " + err.Error())
		}
		i++
		if i%10 == 0 {
			select {
			case <-stop:
				break
			default:
				continue
			}
		}
	}
	duration := time.Since(start)
	s := fmt.Sprintf("%f", duration.Minutes())
	log.Println(tag + " Indexer: batchIndex is done in " + s + " minutes")
}

func (indexer *Indexer) processRealtimeBlock(blockDetail *types.BLockDetail) {
	isRecpRelax := true
	isBatch := false
	if !isMissingRecp(blockDetail) {
		indexer.ProcessBlock(blockDetail, isBatch)
		return
	}
	var block *types.BLockDetail
	blockNumber := blockDetail.BlockNumber
	for i := 0; i < 5; i++ {
		log.Printf("Waiting for 2 more minutes to get receipts for block %v . Trying %v [th] time \n", blockNumber.String(), (i + 1))
		time.Sleep(2 * time.Minute)
		_block, err := indexer.realtimeFetcher.FetchABlock(blockNumber.Int64(), isRecpRelax)
		if err != nil {
			log.Fatalf("Cannot fetch block %v, error: %v \n", blockNumber.String(), err.Error())
		}
		if !isMissingRecp(_block) {
			block = _block
			break
		}
	}
	if block != nil && !isMissingRecp(block) {
		indexer.ProcessBlock(block, isBatch)
		return
	}
	log.Fatalln("Cannot get receipts for block " + blockNumber.String())
}

// ProcessBlock transform blockchain data to our index structure and save it to repo
func (indexer *Indexer) ProcessBlock(blockDetail *types.BLockDetail, isBatch bool) error {
	addressIndex, blockIndex := indexer.CreateIndexData(blockDetail)
	return indexer.IndexRepo.Store(addressIndex, blockIndex, isBatch)
}

// FetchAndProcess fetch a block data from blockchain and process it
func (indexer *Indexer) FetchAndProcess(blockNumber *big.Int) error {
	indexer.createRealtimeFetcher()
	log.Printf("Indexer: Fetching block %v", blockNumber)
	isRecpRelax := false
	blockDetail, err := indexer.realtimeFetcher.FetchABlock(blockNumber.Int64(), isRecpRelax)
	if err != nil {
		return err
	}
	log.Printf("Indexer: Fetched block %v successfully", blockNumber)
	isBatch := true
	err = indexer.ProcessBlock(blockDetail, isBatch)
	if err != nil {
		return err
	}
	log.Printf("Indexer: Processed and saved block %v successfully", blockNumber)
	return err
}

// CreateIndexData transforms blockchain data to our index data
func (indexer *Indexer) CreateIndexData(blockDetail *types.BLockDetail) ([]*types.AddressIndex, *types.BlockIndex) {
	addressIndex := make([]*types.AddressIndex, 0, 2*len(blockDetail.Transactions))
	blockIndex := &types.BlockIndex{
		BlockNumber: blockDetail.BlockNumber.String(),
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
		isNilFrom := false
		from := transaction.From
		if from == "" {
			from = common.AddressZero
			isNilFrom = true
		}

		if !isNilFrom {
			fromIndex := types.AddressIndex{
				TxHash: transaction.TxHash,
				Value:  negValue,
				Time:   blockDetail.Time,
				// BlockNumber:   blockDetail.BlockNumber,
				CoupleAddress: to,
				Status:        transaction.Status,
			}
			if _, ok := sequenceMap[from]; !ok {
				sequenceMap[from] = 0
			}
			sequenceMap[from]++
			fromIndex.Address = from
			fromIndex.Sequence = sequenceMap[from]
			addressIndex = append(addressIndex, &fromIndex)
		}

		if !isNilTo {
			toIndex := types.AddressIndex{
				TxHash: transaction.TxHash,
				Value:  posValue,
				Time:   blockDetail.Time,
				// BlockNumber:   blockDetail.BlockNumber,
				CoupleAddress: from,
				Status:        transaction.Status,
			}
			if _, ok := sequenceMap[to]; !ok {
				sequenceMap[to] = 0
			}
			sequenceMap[to]++
			toIndex.Address = to
			toIndex.Sequence = sequenceMap[to]
			addressIndex = append(addressIndex, &toIndex)
		}
	}
	blockIndex.Addresses = make([]types.AddressSequence, 0, len(sequenceMap))
	for k, v := range sequenceMap {
		blockIndex.Addresses = append(blockIndex.Addresses, types.AddressSequence{Address: k, Sequence: v})
	}
	return addressIndex, blockIndex
}

// GetInitBatches create batch initially
func GetInitBatches(numBatch int, genesisBlock *big.Int, latestBlock *big.Int) []types.BatchStatus {
	result := make([]types.BatchStatus, 0, numBatch)
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

func (indexer *Indexer) createRealtimeFetcher() {
	if indexer.realtimeFetcher != nil {
		return
	}

	fe, err := fetcher.NewChainFetch()
	if err != nil {
		log.Println("Indexer: createRealtimeFetcher failed", err)
		indexer.realtimeFetcher = nil
		return
	}
	indexer.realtimeFetcher = fe
}

func isMissingRecp(blockDetail *types.BLockDetail) bool {
	transactions := blockDetail.Transactions
	for _, tx := range transactions {
		if tx.From == "" && tx.To == "" {
			log.Printf("Indexer: block %v has traansaction %v without receipt \n", blockDetail.BlockNumber.String(), tx.TxHash)
			return true
		}
	}
	return false
}
