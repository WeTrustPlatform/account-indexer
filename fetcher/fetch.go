package fetcher

import (
	"context"
	"log"
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/service"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

//EthClient the Client of geth
type EthClient interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *gethtypes.Header) (ethereum.Subscription, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*gethtypes.Block, error)
	TransactionSender(ctx context.Context, tx *gethtypes.Transaction, block common.Hash, index uint) (common.Address, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*gethtypes.Header, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*gethtypes.Receipt, error)
	TransactionByHash(ctx context.Context, hash common.Hash) (*gethtypes.Transaction, bool, error)
}

// Fetch the interface to interact with blockchain
type Fetch interface {
	RealtimeFetch(ch chan<- *types.BLockDetail)
	FetchABlock(blockNumber *big.Int) (*types.BLockDetail, error)
	GetLatestBlock() (*big.Int, error)
	TransactionByHash(txHash string) (*types.TransactionExtra, error)
}

// ChainFetch the real implementation
type ChainFetch struct {
	Client             EthClient
	blockHeaderChannel chan *gethtypes.Header
}

// NewChainFetch new ChainFetch instance
func NewChainFetch() (*ChainFetch, error) {
	ipcPath := service.GetIpcManager().GetIPC()
	client, err := ethclient.Dial(ipcPath)
	fetcher := &ChainFetch{Client: client}
	if err == nil {
		var sub service.IpcSubscriber = fetcher
		service.GetIpcManager().Subscribe(&sub)
	}
	return fetcher, err
}

// IpcUpdated implements IpcSubscriber interface
func (cf *ChainFetch) IpcUpdated(ipcPath string) {
	// finish any ongoing go-routines of this fetcher
	if cf.blockHeaderChannel != nil {
		// This should finish the RealtimeFetch for loop
		close(cf.blockHeaderChannel)
	}
}

// RealtimeFetch fetch data from blockchain
func (cf *ChainFetch) RealtimeFetch(ch chan<- *types.BLockDetail) {
	ctx := context.Background()
	cf.blockHeaderChannel = make(chan *gethtypes.Header)
	go cf.Client.SubscribeNewHead(ctx, cf.blockHeaderChannel)
	log.Println("RealtimeFetch Waiting for new block hearders...")
	for {
		receivedHeader, ok := <-cf.blockHeaderChannel
		if !ok {
			// switch ipc
			log.Println("Stopping SubscribeNewHead, ipc is switched?")
			break
		}
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
			// Index transactions that create contract too
			if tx.To() == nil && (tx.Value() == nil || tx.Value().Int64() == 0) {
				txRecp, err := cf.Client.TransactionReceipt(ctx, tx.Hash())
				if err == nil {
					if txRecp != nil {
						transaction := types.TransactionDetail{
							From:   "",
							To:     txRecp.ContractAddress.String(),
							TxHash: tx.Hash().String(),
							Value:  tx.Value(),
						}
						transactions = append(transactions, transaction)
					}
				} else {
					log.Printf("Fetch: cannot get receipt for transaction %v, error=%v \n", tx.Hash().String(), err.Error())
				}
			}
		}
	}
	blockDetail := types.BLockDetail{
		BlockNumber:  aBlock.Number(),
		Time:         aBlock.Time(),
		Transactions: transactions,
	}
	return &blockDetail, nil
}

// TransactionByHash query geth node to get addtional data of tx
func (cf *ChainFetch) TransactionByHash(txHash string) (*types.TransactionExtra, error) {
	ctx := context.Background()
	byteArr, err := hexutil.Decode(txHash)
	hash := gethcommon.BytesToHash(byteArr)
	if err != nil {
		return nil, err
	}
	tx, _, err := cf.Client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return &types.TransactionExtra{
		Data:     tx.Data(),
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice(),
	}, nil
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
