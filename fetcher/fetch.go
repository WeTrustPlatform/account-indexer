package fetcher

import (
	"context"
	"fmt"
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

//EthClient the Client of geth
type EthClient interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *gethtypes.Header) (ethereum.Subscription, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*gethtypes.Block, error)
	TransactionSender(ctx context.Context, tx *gethtypes.Transaction, block common.Hash, index uint) (common.Address, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*gethtypes.Header, error)
}

// Fetch the interface to interact with blockchain
type Fetch interface {
	RealtimeFetch(ch chan<- types.BLockDetail)
	FetchABlock(blockNumber *big.Int) (types.BLockDetail, error)
	GetLatestBlock() (*big.Int, error)
}

// ChainFetch the real implementation
type ChainFetch struct {
	Client EthClient
}

// NewChainFetch new ChainFetch instance
func NewChainFetch(ipcPath string) (*ChainFetch, error) {
	client, err := ethclient.Dial(ipcPath)
	return &ChainFetch{Client: client}, err
}

// RealtimeFetch fetch data from blockchain
func (cf *ChainFetch) RealtimeFetch(ch chan<- types.BLockDetail) {
	ctx := context.Background()
	blockHeaderChannel := make(chan *gethtypes.Header)
	go cf.Client.SubscribeNewHead(ctx, blockHeaderChannel)
	fmt.Println("RealtimeFetch Waiting for new block hearders...")
	for {
		receivedHeader := <-blockHeaderChannel
		blockNumber := receivedHeader.Number
		fmt.Println("RealtimeFetch received block number " + blockNumber.String())
		blockDetail, err := cf.FetchABlock(blockNumber)
		if err != nil {
			ch <- blockDetail
		}
	}
}

// FetchABlock fetch a block by block number
func (cf *ChainFetch) FetchABlock(blockNumber *big.Int) (types.BLockDetail, error) {
	ctx := context.Background()
	aBlock, err := cf.Client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		fmt.Println("RealtimeFetch BlockByNumber returns error " + err.Error())
		return types.BLockDetail{}, err
	}
	// fmt.Println(fmt.Sprintf("Found block number received from SubscribeNewHead: %s", blockNumber))
	transactions := []types.TransactionDetail{}
	if len(aBlock.Transactions()) > 0 {
		for index, tx := range aBlock.Transactions() {
			sender, _ := cf.Client.TransactionSender(ctx, tx, aBlock.Hash(), uint(index))
			// fmt.Println(fmt.Sprintf("Hash %s --- To %s --- Value %d -- Sender %s", tx.Hash().String(), tx.To().String(), tx.Value(), sender.String()))
			transaction := types.TransactionDetail{
				From:   sender.String(),
				To:     tx.To().String(),
				TxHash: tx.Hash().String(),
				Value:  *tx.Value(),
			}
			transactions = append(transactions, transaction)
		}
	}
	blockDetail := types.BLockDetail{
		BlockNumber:  *aBlock.Number(),
		Time:         *aBlock.Time(),
		Transactions: transactions,
	}
	return blockDetail, nil
}

// GetLatestBlock get latest known block by geth node
func (cf *ChainFetch) GetLatestBlock() (*big.Int, error) {
	ctx := context.Background()
	// nil means latest known header according to ethclient doc
	header, err := cf.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return big.NewInt(-1), err
	}
	blockNumber := header.Number
	return blockNumber, nil
}
