package http

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/http/types"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/gin-gonic/gin"
)

const (
	DEFAULT_ROWS = 10
)

type HttpServer struct {
	repo repository.Repository
}

func NewServer(repo repository.Repository) HttpServer {
	return HttpServer{repo: repo}
}

// StartServer start http server
func (server HttpServer) Start() {
	router := gin.Default()
	api := router.Group("/api")
	{
		api.GET("/batchstatus", server.getBatchStatus)
		api.GET("/block", server.getBlock)
		api.GET("/account/:accountNumber", server.getTransactionsByAccount)
	}
	// Start and run the server
	router.Run(":3000")
}

func (server HttpServer) getTransactionsByAccount(c *gin.Context) {
	account := c.Param("accountNumber")
	// rows: max result returned
	rowsStr := c.Query("rows")
	// 0-based index
	startStr := c.Query("start")
	rows, err := strconv.Atoi(rowsStr)
	if err != nil {
		rows = DEFAULT_ROWS
	}
	start, err := strconv.Atoi(startStr)
	if err != nil {
		start = 0
	}
	log.Printf("Getting transactions for account %v\n", account)
	total, addressIndexes := server.repo.GetTransactionByAddress(account, rows, start)
	// response automatically marshalled using json.Marshall()
	response := types.EITransactionsByAccount{
		Total:   total,
		Start:   start,
		Indexes: addressIndexes,
	}
	c.JSON(http.StatusOK, response)
}

func (server HttpServer) getBlock(c *gin.Context) {
	blocks := server.repo.GetLastFiveBlocks()
	log.Printf("Number of found blocks : %v \n", len(blocks))
	response := map[string]string{}
	for _, block := range blocks {
		response[block.BlockNumber] = fmt.Sprintf("%v", block.Addresses)
	}
	c.JSON(http.StatusOK, response)
	// blockNumber := c.Param("blockNumber")
	// if blockNumber == nil {
	// 	// Get 5 last block
	// } else if blockNumber == "latest" {
	// 	// TODO
	// } else {
	// 	// TODO: get a specific block number
	// }
}

func (server HttpServer) getBatchStatus(c *gin.Context) {
	batchStatuses := server.repo.GetAllBatchStatuses()
	response := map[string]string{}
	for _, batch := range batchStatuses {
		current := ""
		if batch.Current != nil {
			current = batch.Current.String()
		}
		key := fmt.Sprintf("From %v, To %v", batch.From.String(), batch.To.String())
		updatedAt := common.UnmarshallIntToTime(batch.UpdatedAt)
		value := fmt.Sprintf("Current %v, Updated At %v", current, updatedAt)
		response[key] = value
	}
	c.JSON(http.StatusOK, response)
}
