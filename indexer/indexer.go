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

type Range struct {
	// Inclusive
	From *big.Int
	// Exclusive
	To *big.Int
}

// DivideRange performance states that having > 2 goroutines have same performance
// so let's go with 2 goroutines
func DivideRange(parent Range) (Range, Range) {
	minusFrom := new(big.Int)
	minusFrom = minusFrom.Neg(parent.From)
	distance := new(big.Int)
	distance = distance.Add(parent.To, minusFrom)
	distance = distance.Div(distance, big.NewInt(2))
	middle := new(big.Int)
	middle = middle.Add(parent.From, distance)
	toPlus1 := new(big.Int)
	toPlus1 = toPlus1.Add(parent.To, big.NewInt(1))
	range1From := new(big.Int)
	range1From = range1From.Set(parent.From)
	range1 := Range{From: range1From, To: middle}
	range2 := Range{From: middle, To: toPlus1}
	return range1, range2
}

// RealtimeIndex entry point for this struct
func (indexer *Indexer) RealtimeIndex() {
	fetcher, err := fetcher.NewChainFetch(indexer.IpcPath)
	if err != nil {
		log.Fatal("Can't connect to IPC server", err)
		return
	}
	indexerChannel := make(chan types.BLockDetail)
	// go indexer.Fetcher.RealtimeFetch(indexerChannel)
	go fetcher.RealtimeFetch(indexerChannel)
	for {
		blockDetail := <-indexerChannel
		fmt.Println("indexer: Received BlockDetail " + blockDetail.BlockNumber.String())
		indexer.processBlock(blockDetail)
	}
}

// IndexFromGenesis index from block 1
func (indexer *Indexer) IndexFromGenesis() {
	// TODO: change this latest block to realtime?
	// latestBlock := big.NewInt(7000000)
	latestBlock := big.NewInt(100000)
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
	from := rg.From
	to := rg.To
	fetcher, err := fetcher.NewChainFetch(indexer.IpcPath)
	if err != nil {
		log.Fatal("Can't connect to IPC server", err)
		return
	}
	blockNumber := new(big.Int)
	for blockNumber.Set(from); blockNumber.Cmp(to) < 0; blockNumber = blockNumber.Add(blockNumber, big.NewInt(int64(1))) {
		// fmt.Println("indexer: Received BlockDetail " + blockNumber.String())
		blockDetail, err := fetcher.FetchABlock(blockNumber)
		if err == nil {
			fmt.Println(tag + " indexer: Received BlockDetail " + blockDetail.BlockNumber.String())
			indexer.processBlock(blockDetail)
		} else {
			fmt.Println(tag + " indexer: cannot get block " + blockNumber.String() + " , error is " + err.Error())
			// TODO: log warning
		}
	}
	fmt.Println(tag + " is done indexByRange from=" + from.String())
}

func (indexer *Indexer) processBlock(blockDetail types.BLockDetail) {
	addressIndex, blockIndex := indexer.CreateIndexData(blockDetail)
	indexer.Repo.Store(addressIndex, blockIndex)
	fmt.Println("indexer: Saved block " + blockDetail.BlockNumber.String() + " to Repository already")
}

// CreateIndexData transforms blockchain data to our index data
func (indexer *Indexer) CreateIndexData(blockDetail types.BLockDetail) ([]types.AddressIndex, types.BlockIndex) {
	addressIndex := make([]types.AddressIndex, 0, 2*len(blockDetail.Transactions))
	blockIndex := types.BlockIndex{
		BlockNumber: blockDetail.BlockNumber.String(),
		Addresses:   []string{},
	}
	tmpMap := map[string]bool{}

	for _, transaction := range blockDetail.Transactions {
		// TODO: resolve pointer issue
		posValue := transaction.Value
		negValue := transaction.Value.Mul(&posValue, big.NewInt(-1))

		fromIndex := types.AddressIndex{
			Address:       transaction.From,
			TxHash:        transaction.TxHash,
			Value:         *negValue,
			Time:          blockDetail.Time,
			BlockNumber:   blockDetail.BlockNumber,
			CoupleAddress: transaction.To,
		}

		toIndex := types.AddressIndex{
			Address:       transaction.To,
			TxHash:        transaction.TxHash,
			Value:         posValue,
			Time:          blockDetail.Time,
			BlockNumber:   blockDetail.BlockNumber,
			CoupleAddress: transaction.From,
		}
		if !tmpMap[transaction.From] {
			tmpMap[transaction.From] = true
			blockIndex.Addresses = append(blockIndex.Addresses, transaction.From)
		}
		if !tmpMap[transaction.To] {
			tmpMap[transaction.To] = true
			blockIndex.Addresses = append(blockIndex.Addresses, transaction.To)
		}
		addressIndex = append(addressIndex, fromIndex, toIndex)
	}
	return addressIndex, blockIndex
}

// func Index111(ipcPath string) {
// 	fmt.Println("Hello from Tuyen")
// 	client, err := ethclient.Dial(ipcPath)
// 	if err != nil {
// 		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
// 	} else {
// 		fmt.Println("Successfully connect to geth ipc")
// 	}
// 	ctx := context.Background()

// 	blockHash := "0x96a3e88d84bc3538f2d9e61fbcab154a227d51e586fb4f7d404105257bcf0277"
// 	aBlock, err := client.BlockByHash(ctx, common.HexToHash(blockHash))
// 	if err != nil {
// 		log.Fatal(fmt.Sprintf("Failed to get block %s", blockHash))
// 	} else {
// 		fmt.Println(fmt.Sprintf("Block number of %s: %s", blockHash, aBlock.Number()))
// 	}

// 	address := "0xa7046014126a840e0420c77196982fb7499f9778"
// 	amount, err := client.BalanceAt(ctx, common.HexToAddress(address), nil)
// 	if err != nil {
// 		log.Fatal("Failed to get balance of address")
// 	} else {
// 		fmt.Println(fmt.Sprintf("Balance of address %s: %s", address, amount))
// 	}
// 	// latest block
// 	aBlock, err = client.BlockByNumber(ctx, nil)
// 	if err != nil {
// 		log.Fatal("Failed to get balance of address")
// 	} else {
// 		fmt.Println(fmt.Sprintf("Latest block number: %s", aBlock.Number()))
// 	}

// 	latestBlock := aBlock.Number()
// 	for i := big.NewInt(int64(1)); i.Cmp(latestBlock) < 0; i = i.Add(i, big.NewInt(int64(1))) {
// 		// aBlock = client.
// 		printBlockDetail(client, ctx, i)
// 	}

// 	blockHeaderChannel := make(chan *types.Header)
// 	client.SubscribeNewHead(ctx, blockHeaderChannel)
// 	fmt.Println("Waiting for new block hearders...")
// 	for {
// 		receivedHeader := <-blockHeaderChannel
// 		blockNumber := receivedHeader.Number
// 		fmt.Println(fmt.Sprintf("Found block number received from SubscribeNewHead: %s", blockNumber))
// 		printBlockDetail(client, ctx, blockNumber)
// 	}

// }

// func printBlockDetail(client *ethclient.Client, ctx context.Context, i *big.Int) {
// 	aBlock, _ := client.BlockByNumber(ctx, i)
// 	if len(aBlock.Transactions()) > 0 {
// 		fmt.Println(fmt.Sprintf("There are transactions in block %s", aBlock.Number()))
// 		for index, tx := range aBlock.Transactions() {
// 			sender, _ := client.TransactionSender(ctx, tx, aBlock.Hash(), uint(index+100))
// 			fmt.Println(fmt.Sprintf("Hash %s --- To %s --- Value %d -- Sender %s", tx.Hash().String(), tx.To().String(), tx.Value(), sender.String()))
// 		}
// 	}
// }
