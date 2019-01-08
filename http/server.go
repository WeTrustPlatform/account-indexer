package http

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/http/types"
	"github.com/WeTrustPlatform/account-indexer/repository"
	"github.com/gin-gonic/gin"
)

const (
	DEFAULT_ROWS    = 10
	ADMIN_USER_NAME = "userName"
	ADMIN_PASSWORD  = "password"
)

type HttpServer struct {
	repo repository.Repository
}

func NewServer(repo repository.Repository) HttpServer {
	return HttpServer{repo: repo}
}

// Start start http server
func (server HttpServer) Start() {
	router := gin.Default()
	api := router.Group("/api")
	{
		api.GET("/account/:accountNumber", server.getTransactionsByAccount)
	}

	admin := router.Group("/admin", gin.BasicAuth(gin.Accounts{
		os.Getenv(ADMIN_USER_NAME): os.Getenv(ADMIN_PASSWORD),
	}))
	{
		admin.GET("/batch/status", server.getBatchStatus)
		// admin.POST("/batch/restart", server.restartBatch)
		admin.GET("/block/:blockNumber", server.getBlock)
		admin.GET("/block", server.getBlock)
	}
	// Start and run the server
	router.Run(":3000")
}

func (server HttpServer) getTransactionsByAccount(c *gin.Context) {
	account := c.Param("accountNumber")
	rows, start := getPagingQueryParams(c)
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
	blockNumber := c.Param("blockNumber")
	rows, start := getPagingQueryParams(c)
	total, blocks := server.repo.GetBlocks(blockNumber, rows, start)
	response := types.EIBlocks{
		Total:   total,
		Start:   start,
		Indexes: blocks,
	}
	log.Printf("Number of found blocks : %v \n", len(blocks))
	c.JSON(http.StatusOK, response)
}

func (server HttpServer) getBatchStatus(c *gin.Context) {
	batchStatuses := server.repo.GetAllBatchStatuses()
	response := []types.EIBatchStatus{}
	for _, batch := range batchStatuses {
		current := ""
		if batch.Current != nil {
			current = batch.Current.String()
		}
		eiBatch := types.EIBatchStatus{
			From:      batch.From,
			To:        batch.To,
			Current:   current,
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
		rows = DEFAULT_ROWS
	}
	start, err := strconv.Atoi(startStr)
	if err != nil {
		start = 0
	}
	return rows, start
}
