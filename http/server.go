package http

import (
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/http/types"
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

// HttpServer http server
type HttpServer struct {
	indexRepo repository.IndexRepo
	batchRepo repository.BatchRepo
}

// NewServer Rest API
func NewServer(indexRepo repository.IndexRepo, batchRepo repository.BatchRepo) HttpServer {
	return HttpServer{indexRepo: indexRepo, batchRepo: batchRepo}
}

// Start start http server
func (server HttpServer) Start() {
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

func (server HttpServer) getTransactionsByAccount(c *gin.Context) {
	account := c.Param("accountNumber")
	accountByteArr, err := hexutil.Decode(account)
	if err != nil || len(accountByteArr) == 0 {
		c.JSON(400, gin.H{"msg": "invalid account " + account})
		return
	}
	fromTimeStr := c.Query("fromTime")
	var fromTime *big.Int
	if len(fromTimeStr) > 0 {
		fromTime, err = common.StrToUnixTimeInt(fromTimeStr)
		if err != nil {
			c.JSON(400, gin.H{"msg": "invalid fromTime " + fromTimeStr})
			return
		}
	}

	toTimeStr := c.Query("toTime")
	var toTime *big.Int
	if len(toTimeStr) > 0 {
		toTime, err = common.StrToUnixTimeInt(toTimeStr)
		if err != nil {
			c.JSON(400, gin.H{"msg": "invalid toTime " + toTimeStr})
			return
		}
	}

	rows, start := getPagingQueryParams(c)
	log.Printf("Getting transactions for account %v\n", account)
	total, addressIndexes := server.indexRepo.GetTransactionByAddress(account, rows, start, fromTime, toTime)
	// response automatically marshalled using json.Marshall()
	response := types.EITransactionsByAccount{
		Total:   total,
		Start:   start,
		Indexes: addressIndexes,
	}
	c.JSON(http.StatusOK, response)
}

func (server HttpServer) getBlock(c *gin.Context) {
	blockNumber := c.Param("blockNumber")
	rows, start := getPagingQueryParams(c)
	total, blocks := server.indexRepo.GetBlocks(blockNumber, rows, start)
	response := types.EIBlocks{
		Total:   total,
		Start:   start,
		Indexes: blocks,
	}
	log.Printf("Number of found blocks : %v \n", len(blocks))
	c.JSON(http.StatusOK, response)
}

func (server HttpServer) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, common.GetConfig().String())
}

func (server HttpServer) getBatchStatus(c *gin.Context) {
	batchStatuses := server.batchRepo.GetAllBatchStatuses()
	response := []types.EIBatchStatus{}
	for _, batch := range batchStatuses {
		current := ""
		if batch.Current != nil {
			current = batch.Current.String()
		}
		eiBatch := types.EIBatchStatus{
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
