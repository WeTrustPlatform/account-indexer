package http

import (
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/common/config"
	"github.com/WeTrustPlatform/account-indexer/fetcher"
	httpTypes "github.com/WeTrustPlatform/account-indexer/http/types"
	"github.com/WeTrustPlatform/account-indexer/indexer"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/WeTrustPlatform/account-indexer/service"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
)

const (
	// DefaultRows Default row for paging
	DefaultRows = 10
	// AdminUserName Username for http admin api
	AdminUserName = "INDEXER_USER_NAME"
	// AdminPassword Password for http admin api
	AdminPassword = "INDEXER_PASSWORD"
)

// Server http server
type Server struct {
	indexRepo repository.IndexRepo
	batchRepo repository.BatchRepo
	indexer   indexer.Indexer
	fetcher   fetcher.Fetch
}

// NewServer Rest API
func NewServer(idx indexer.Indexer) Server {

	indexRepo := idx.IndexRepo
	batchRepo := idx.BatchRepo

	server := Server{indexRepo: indexRepo, batchRepo: batchRepo, indexer: idx}
	var sub service.IpcSubscriber = &server
	service.GetIpcManager().Subscribe(&sub)
	// Don't care the error, if there is error then IPCUpdate will call
	fetcher, err := fetcher.NewChainFetch()
	if err != nil {
		log.Println("Server: cannot create fetcher err=", err)
	} else {
		server.fetcher = fetcher
	}
	return server
}

// IpcUpdated implements IpcSubscriber interface
func (server *Server) IpcUpdated(ipcPath string) {
	// Don't care the error, if there is error then IPCUpdate will call
	fetcher, err := fetcher.NewChainFetch()
	if err != nil {
		log.Println("Server - IPCUpdated: cannot create net fetcher err=", err)
		return
	}
	log.Println("Server - IPCUpdated: update to new IPC successfully")
	server.fetcher = fetcher
}

// Name implements IpcSubscriber interface
func (server *Server) Name() string {
	return "Server"
}

// Start start http server
func (server *Server) Start() {
	router := gin.Default()
	api := router.Group("/api")
	{
		api.GET("v1/accounts/:accountNumber", server.getTransactionsByAccount)
		api.GET("v1/accounts/:accountNumber/total", server.getTotalByAccount)
	}

	admin := router.Group("/admin", gin.BasicAuth(gin.Accounts{
		os.Getenv(AdminUserName): os.Getenv(AdminPassword),
	}))
	{
		admin.GET("/batches/status", server.getBatchStatus)
		// admin.POST("/batch/restart", server.restartBatch)
		admin.GET("/blocks/:blockNumber", server.getBlock)
		admin.POST("/blocks/:blockNumber", server.rerunBlock)
		admin.GET("/blocks", server.getBlock)
		admin.GET("/config", server.getConfig)
		admin.GET("/version", server.getVersion)
	}
	// Listen for port 3000 on localhost(127.0.0.1)
	// Admin needs to setup a reversed proxy and forward to http://127.0.0.1:3000
	err := router.Run(fmt.Sprintf("127.0.0.1:%v", config.GetConfig().Port))
	if err == nil {
		log.Println("Server: Started server successfully at port ", config.GetConfig().Port)
	} else {
		log.Fatal("Server: Cannot start http server", err.Error())
	}
}

func (server *Server) getTransactionsByAccount(c *gin.Context) {
	account, fromTime, toTime, err := getAccountParam(c)
	if err != nil {
		return
	}
	flParam := c.Query("fl")
	addlFields := strings.Split(flParam, ",")
	needTxData := common.Contains(addlFields, "data")
	needGas := common.Contains(addlFields, "gas")
	needGasPrice := common.Contains(addlFields, "gasPrice")

	rows, start := getPagingQueryParams(c)
	log.Printf("Server: Getting transactions for account %v\n", account)
	total, addressIndexes := server.indexRepo.GetTransactionByAddress(account, rows, start, fromTime, toTime)
	addresses := []httpTypes.EIAddress{}
	for _, idx := range addressIndexes {
		addr := httpTypes.AddressToEIAddress(idx)
		if needTxData || needGas || needGasPrice {
			addlTxData, err := server.fetcher.TransactionByHash(addr.TxHash)
			if err == nil {
				if needTxData {
					addr.Data = addlTxData.Data
				}
				if needGas {
					addr.Gas = addlTxData.Gas
				}
				if needGasPrice {
					addr.GasPrice = addlTxData.GasPrice
				}

			} else {
				log.Println("Server: Warning: cannot get additional data for transaction ", addr.TxHash)
			}
		}
		addresses = append(addresses, addr)
	}
	totalStr := strconv.Itoa(total)
	if total > common.NumMaxTransaction {
		// If this address has a lot of transactions, just say +10000
		totalStr = "+" + strconv.Itoa(common.NumMaxTransaction)
	}
	// response automatically marshalled using json.Marshall()
	response := httpTypes.EITransactionsByAccount{
		Total:   totalStr,
		Start:   start,
		Indexes: addresses,
	}
	c.JSON(http.StatusOK, response)
}

func (server *Server) getTotalByAccount(c *gin.Context) {
	account, fromTime, toTime, err := getAccountParam(c)
	if err != nil {
		return
	}
	total := server.indexRepo.GetTotalTransaction(account, fromTime, toTime)
	response := httpTypes.EITotalTransaction{
		Total: total,
	}
	c.JSON(http.StatusOK, response)
}

func (server *Server) getBlock(c *gin.Context) {
	blockNumber := c.Param("blockNumber")
	rows, start := getPagingQueryParams(c)
	total, blocks := server.indexRepo.GetBlocks(blockNumber, rows, start)
	response := httpTypes.EIBlocks{
		Total:   total,
		Start:   start,
		Indexes: blocks,
	}
	log.Printf("Server: Number of found blocks : %v \n", len(blocks))
	c.JSON(http.StatusOK, response)
}

func (server *Server) rerunBlock(c *gin.Context) {
	blockNumberStr := c.Param("blockNumber")
	log.Printf("Server: Getting block %v from http param", blockNumberStr)
	blockNumber := new(big.Int)
	blockNumber, ok := blockNumber.SetString(blockNumberStr, 10)
	if !ok {
		c.JSON(400, gin.H{"msg": "invalid block nunmber " + blockNumberStr})
		return
	}

	err := server.indexer.FetchAndProcess(blockNumber)
	if err != nil {
		c.JSON(500, gin.H{"msg": "internal server error " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, fmt.Sprintf("Indexer processed block %v successfully", blockNumberStr))
}

func (server *Server) getConfig(c *gin.Context) {
	ipc := service.GetIpcManager().GetIPC()
	c.JSON(http.StatusOK, config.GetConfig().String()+" ipc="+ipc)
}

func (server *Server) getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, config.GetVersion())
}

func (server *Server) getBatchStatus(c *gin.Context) {
	batchStatuses := server.batchRepo.GetAllBatchStatuses()
	response := []httpTypes.EIBatchStatus{}
	for _, batch := range batchStatuses {
		current := ""
		if batch.Current != nil {
			current = batch.Current.String()
		}
		eiBatch := httpTypes.EIBatchStatus{
			From:      batch.From,
			To:        batch.To,
			Step:      batch.Step,
			Current:   current,
			CreatedAt: common.UnmarshallIntToTime(batch.CreatedAt),
			UpdatedAt: common.UnmarshallIntToTime(batch.UpdatedAt),
		}
		response = append(response, eiBatch)
	}
	c.JSON(http.StatusOK, response)
}

// Return rows, start http query params
func getPagingQueryParams(c *gin.Context) (int, int) {
	// rows: max result returned
	rowsStr := c.Query("rows")
	// 0-based index
	startStr := c.Query("start")
	rows, err := strconv.Atoi(rowsStr)
	if err != nil {
		rows = DefaultRows
	}
	start, err := strconv.Atoi(startStr)
	if err != nil {
		start = 0
	}
	return rows, start
}

// Get and validate account, from, to
func getAccountParam(c *gin.Context) (string, time.Time, time.Time, error) {
	var fromTime time.Time
	var toTime time.Time
	account := c.Param("accountNumber")
	accountByteArr, err := hexutil.Decode(account)
	if err != nil || len(accountByteArr) == 0 {
		c.JSON(400, gin.H{"msg": "invalid account " + account})
		return account, fromTime, toTime, err
	}
	fromTimeStr := c.Query("from")

	if len(fromTimeStr) > 0 {
		fromTime, err = common.StrToTime(fromTimeStr)
		if err != nil {
			c.JSON(400, gin.H{"msg": "invalid from " + fromTimeStr})
			return account, fromTime, toTime, err
		}
	}

	toTimeStr := c.Query("to")
	if len(toTimeStr) > 0 {
		toTime, err = common.StrToTime(toTimeStr)
		if err != nil {
			c.JSON(400, gin.H{"msg": "invalid to " + toTimeStr})
			return account, fromTime, toTime, err
		}
	}
	return account, fromTime, toTime, nil
}
