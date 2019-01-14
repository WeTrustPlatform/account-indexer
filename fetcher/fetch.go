package fetcher

import (
	"context"
	"log"
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/service"
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
	RealtimeFetch(ch chan<- *types.BLockDetail)
	FetchABlock(blockNumber *big.Int) (*types.BLockDetail, error)
	GetLatestBlock() (*big.Int, error)
}

// ChainFetch the real implementation
type ChainFetch struct {
	Client EthClient
}

// NewChainFetch new ChainFetch instance
func NewChainFetch() (*ChainFetch, error) {
	ipcPath := service.GetIpcManager().GetIPC()
	client, err := ethclient.Dial(ipcPath)
	fetcher := &ChainFetch{Client: client}
	if err != nil {
		var sub service.IpcSubscriber
		sub = fetcher
		service.GetIpcManager().Subscribe(&sub)
	}
	return fetcher, err
}

// IpcUpdated implement IpcSubscriber interface
func (cf *ChainFetch) IpcUpdated(ipcPath string) {
	client, err := ethclient.Dial(ipcPath)
	if err != nil {
		log.Fatal("Not able to switch to ipc", ipcPath, err.Error())
	}
	cf.Client = client
	log.Println("Switched to new IPC", ipcPath)
}

// RealtimeFetch fetch data from blockchain
func (cf *ChainFetch) RealtimeFetch(ch chan<- *types.BLockDetail) {
	ctx := context.Background()
	blockHeaderChannel := make(chan *gethtypes.Header)
	go cf.Client.SubscribeNewHead(ctx, blockHeaderChannel)
	log.Println("RealtimeFetch Waiting for new block hearders...")
	for {
		receivedHeader := <-blockHeaderChannel
		blockNumber := receivedHeader.Number
		// log.Println("RealtimeFetch received block number " + blockNumber.String())
		blockDetail, err := cf.FetchABlock(blockNumber)
		if err == nil {
			ch <- blockDetail
		} else {
			log.Fatal("Cannot get block detail for block " + blockNumber.String())
		}
	}
}

// FetchABlock fetch a block by block number
func (cf *ChainFetch) FetchABlock(blockNumber *big.Int) (*types.BLockDetail, error) {
	ctx := context.Background()
	aBlock, err := cf.Client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		log.Fatal("RealtimeFetch BlockByNumber returns error " + err.Error())
		return &types.BLockDetail{}, err
	}
	// log.Println(fmt.Sprintf("Found block number received from SubscribeNewHead: %s", blockNumber))
	transactions := []types.TransactionDetail{}
	if len(aBlock.Transactions()) > 0 {
		for index, tx := range aBlock.Transactions() {
			sender, _ := cf.Client.TransactionSender(ctx, tx, aBlock.Hash(), uint(index))
			// log.Println(fmt.Sprintf("Hash %s --- Value %d -- Sender %s", tx.Hash().String(), tx.Value(), sender.String()))
			// Some transactions have nil To, for example Contract creation
			to := ""
			if tx.To() != nil {
				to = tx.To().String()
			}
			transaction := types.TransactionDetail{
				From:   sender.String(),
				To:     to,
				TxHash: tx.Hash().String(),
				Value:  tx.Value(),
			}
			transactions = append(transactions, transaction)
		}
	}
	blockDetail := types.BLockDetail{
		BlockNumber:  aBlock.Number(),
		Time:         aBlock.Time(),
		Transactions: transactions,
	}
	return &blockDetail, nil
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
