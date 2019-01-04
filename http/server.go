package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/gin-gonic/gin"
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
	fmt.Printf("Getting transactions for account %v\n", account)
	addressIndexes := server.repo.GetTransactionByAddress(account)
	response := map[string]string{}
	for _, index := range addressIndexes {
		txTime := common.UnmarshallIntToTime(&index.Time)
		response[index.TxHash] = fmt.Sprintf("Value: %v, Time: %v, BlockNumber: %v, CoupleAddress: %v", index.Value.String(), txTime, index.BlockNumber.String(), index.CoupleAddress)
	}
	c.JSON(http.StatusOK, response)
}

func (server HttpServer) getBlock(c *gin.Context) {
	blocks := server.repo.GetLastFiveBlocks()
	fmt.Printf("Number of found blocks : %v \n", len(blocks))
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
		updatedAt := time.Unix(0, int64(batch.UpdatedAt.Uint64()))
		value := fmt.Sprintf("Current %v, Updated At %v", current, updatedAt)
		response[key] = value
	}
	c.JSON(http.StatusOK, response)
}
