package fetcher

import (
	"context"
	"log"
	"math/big"
	"time"

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
	Close()
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
	ethSub             ethereum.Subscription
}

// NewChainFetch new ChainFetch instance
func NewChainFetch() (*ChainFetch, error) {
	ipcPath := service.GetIpcManager().GetIPC()
	client, err := ethclient.Dial(ipcPath)
	if err != nil {
		log.Printf("ChainFetch: Cannot dial, ipc %v is wrong, error: %v", ipcPath, err.Error())
		switchIPC()
		return nil, err
	}
	fetcher := &ChainFetch{Client: client}
	fetcher.blockHeaderChannel = nil
	return fetcher, err
}

// IpcUpdated no need to implement IpcSubscriber interface
// Let indexer update me
func (cf *ChainFetch) IpcUpdated() {
	// finish any ongoing go-routines of this fetcher
	if cf.ethSub != nil {
		cf.ethSub.Unsubscribe()
	}
	log.Println("ChainFetch: Unsubscribed the old ipc, wait for 30s")
	time.Sleep(30 * time.Second)
	log.Println("ChainFetch: 30s passed, closing blockHeaderChannel")
	// writer should be the one to close the channel?
	if cf.blockHeaderChannel != nil {
		// This should finish the RealtimeFetch for loop
		close(cf.blockHeaderChannel)
	}
	cf.Client.Close()
}

// RealtimeFetch fetch data from blockchain
func (cf *ChainFetch) RealtimeFetch(ch chan<- *types.BLockDetail) {
	// don't subscribe ever, let indexer do it
	ctx := context.Background()
	cf.blockHeaderChannel = make(chan *gethtypes.Header)
	ethSub, err := cf.Client.SubscribeNewHead(ctx, cf.blockHeaderChannel)
	if err != nil {
		log.Printf("ChainFetch: Cannot do newHead subscribe to this ipc %v\n", service.GetIpcManager().GetIPC())
		switchIPC()
		return
	}
	cf.ethSub = ethSub

	log.Println("ChainFetch: RealtimeFetch Waiting for new block hearders...")
	for {
		receivedHeader, ok := <-cf.blockHeaderChannel
		if !ok {
			// switched ipc
			log.Println("ChainFetch: Stopping SubscribeNewHead, ipc is switched?")
			close(ch)
			break
		}
		blockNumber := receivedHeader.Number
		blockDetail, err := cf.FetchABlock(blockNumber)
		if err == nil {
			ch <- blockDetail
		} else {
			// Finish the Realtime process, someone will switch the IPC
			log.Println("ChainFetch: RealtimeFetch Cannot get block detail for block " + blockNumber.String())
			close(ch)
			break
		}
	}
	log.Println("ChainFetch: Stopped RealtimeFetch")
}

// FetchABlock fetch a block by block number
func (cf *ChainFetch) FetchABlock(blockNumber *big.Int) (*types.BLockDetail, error) {
	ctx := context.Background()
	aBlock, err := cf.Client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		log.Println("ChainFetch: FetchABlock BlockByNumber returns error " + err.Error())
		switchIPC()
		return &types.BLockDetail{}, err
	}
	transactions := []types.TransactionDetail{}
	if len(aBlock.Transactions()) > 0 {
		for index, tx := range aBlock.Transactions() {
			sender, err := cf.Client.TransactionSender(ctx, tx, aBlock.Hash(), uint(index))
			if err != nil {
				log.Println("ChainFetch: FetchABlock TransactionSender returns error " + err.Error())
				switchIPC()
				return &types.BLockDetail{}, err
			}
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
					log.Printf("ChainFetch: FetchABlock warning cannot get receipt for transaction %v, error=%v \n", tx.Hash().String(), err.Error())
					// https://github.com/WeTrustPlatform/account-indexer/issues/18
					// We trust our geth nodes, some other nodes always return error for this API
					// switchIPC()
					// return &types.BLockDetail{}, err
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
		switchIPC()
		return big.NewInt(-1), err
	}
	blockNumber := header.Number
	return blockNumber, nil
}

func switchIPC() {
	go service.GetIpcManager().ForceChangeIPC()
}
