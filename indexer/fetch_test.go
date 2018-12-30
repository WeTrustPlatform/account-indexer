package indexer

import (
	"context"
	"math/big"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethCommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/tuyennhv/geth-indexer/core/types"
)

type MockEthClient struct{}

var header = &gethtypes.Header{
	Number: big.NewInt(2019),
}

var fromStr = "address From"
var toStr = "address To"
var from = gethCommon.BytesToAddress([]byte(fromStr))
var to = gethCommon.BytesToAddress([]byte(toStr))

var amount = big.NewInt(100)

var transactions = []*gethtypes.Transaction{
	gethtypes.NewTransaction(uint64(1), to, amount, uint64(21000), big.NewInt(100), nil),
}

func (mec MockEthClient) SubscribeNewHead(ctx context.Context, ch chan<- *gethtypes.Header) (ethereum.Subscription, error) {
	go func() {
		ch <- header
	}()
	return nil, nil
}

func (mec MockEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*gethtypes.Block, error) {
	block := gethtypes.NewBlockWithHeader(header).WithBody(transactions, nil)
	return block, nil
}

func (mec MockEthClient) TransactionSender(ctx context.Context, tx *gethtypes.Transaction, block common.Hash, index uint) (common.Address, error) {
	return from, nil
}

func TestFetchData(t *testing.T) {
	fetcher := ChainFetch{
		Client: MockEthClient{},
	}
	indexerChannel := make(chan types.BLockDetail)
	go fetcher.RealtimeFetch(indexerChannel)
	time.Sleep(time.Second * 1)
	blockDetail := <-indexerChannel
	if blockDetail.BlockNumber.Uint64() != header.Number.Uint64() {
		t.Error("Test failed: Block number is not the same:", blockDetail.BlockNumber, header.Number)
	}
	transaction := blockDetail.Transactions[0]
	if transaction.From != from.String() {
		t.Error("Test failed, from address is not correct: ", transaction.From, from.String())
	}
	if transaction.To != to.String() {
		t.Error("Test failed, to address is not correct: ", transaction.To, to.String())
	}
}
