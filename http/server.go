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
	"github.com/WeTrustPlatform/account-indexer/fetcher"
	httpTypes "github.com/WeTrustPlatform/account-indexer/http/types"
	"github.com/WeTrustPlatform/account-indexer/indexer"
	"github.com/WeTrustPlatform/account-indexer/repository"
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
	fetcher, err := fetcher.NewChainFetch()
	if err != nil {
		log.Fatal("IPC path is not correct error:", err.Error())
	}
	return Server{indexRepo: indexRepo, batchRepo: batchRepo, indexer: idx, fetcher: fetcher}
}

// Start start http server
func (server Server) Start() {
	router := gin.Default()
	api := router.Group("/api")
	{
		api.GET("v1/accounts/:accountNumber", server.getTransactionsByAccount)
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
	}
	// Start and run the server
	err := router.Run(fmt.Sprintf(":%v", common.GetConfig().Port))
	if err == nil {
		log.Println("Started server successfully at port ", common.GetConfig().Port)
	} else {
		log.Fatal("Cannot start http server", err.Error())
	}
}

func (server Server) getTransactionsByAccount(c *gin.Context) {
	account := c.Param("accountNumber")
	accountByteArr, err := hexutil.Decode(account)
	if err != nil || len(accountByteArr) == 0 {
		c.JSON(400, gin.H{"msg": "invalid account " + account})
		return
	}
	fromTimeStr := c.Query("from")
	var fromTime time.Time
	if len(fromTimeStr) > 0 {
		fromTime, err = common.StrToTime(fromTimeStr)
		if err != nil {
			c.JSON(400, gin.H{"msg": "invalid from " + fromTimeStr})
			return
		}
	}

	toTimeStr := c.Query("to")
	var toTime time.Time
	if len(toTimeStr) > 0 {
		toTime, err = common.StrToTime(toTimeStr)
		if err != nil {
			c.JSON(400, gin.H{"msg": "invalid to " + toTimeStr})
			return
		}
	}

	flParam := c.Query("fl")
	addlFields := strings.Split(flParam, ",")
	needTxData := common.Contains(addlFields, "data")
	needGas := common.Contains(addlFields, "gas")
	needGasPrice := common.Contains(addlFields, "gasPrice")

	rows, start := getPagingQueryParams(c)
	log.Printf("Getting transactions for account %v\n", account)
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
				log.Println("Warning: cannot get additional data for transaction ", addr.TxHash)
			}
		}
		addresses = append(addresses, addr)
	}
	// response automatically marshalled using json.Marshall()
	response := httpTypes.EITransactionsByAccount{
		Total:   total,
		Start:   start,
		Indexes: addresses,
	}
	c.JSON(http.StatusOK, response)
}

func (server Server) getBlock(c *gin.Context) {
	blockNumber := c.Param("blockNumber")
	rows, start := getPagingQueryParams(c)
	total, blocks := server.indexRepo.GetBlocks(blockNumber, rows, start)
	response := httpTypes.EIBlocks{
		Total:   total,
		Start:   start,
		Indexes: blocks,
	}
	log.Printf("Number of found blocks : %v \n", len(blocks))
	c.JSON(http.StatusOK, response)
}

func (server Server) rerunBlock(c *gin.Context) {
	blockNumberStr := c.Param("blockNumber")
	log.Printf("Getting block %v from http param", blockNumberStr)
	blockNumber := new(big.Int)
	blockNumber, ok := blockNumber.SetString(blockNumberStr, 10)
	if ok == false {
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

func (server Server) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, common.GetConfig().String())
}

func (server Server) getBatchStatus(c *gin.Context) {
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
