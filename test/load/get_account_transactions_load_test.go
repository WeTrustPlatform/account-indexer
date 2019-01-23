// +build load

package load

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	httpTypes "github.com/WeTrustPlatform/account-indexer/http/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

/**
 * This test is not expected to run in travis.
 * Change this URL to test a different server
 */
const IndexerUrl = "http://mainnet.kivutar.me:3000/api/v1/accounts/%v"
const EthereumNode = "wss://mainnet.infura.io/_ws"
const BlockCount = 3
const StartBlock = 7000000

func getClient() *http.Client {
	// Customize the Transport to have larger connection pool
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport := *defaultTransportPointer // dereference it to get a copy of the struct that the pointer points to
	defaultTransport.MaxIdleConns = 100
	defaultTransport.MaxIdleConnsPerHost = 100

	myClient := &http.Client{
		Transport: &defaultTransport,
		Timeout:   100 * time.Second,
	}
	return myClient
}

func TestHandleMultipleRequest(t *testing.T) {
	addresses := getAddressesFromBlockchain()
	log.Printf("Found %v address from block %v to block %v \n", len(addresses), StartBlock, (StartBlock + BlockCount - 1))
	now := time.Now()
	mainWG := sync.WaitGroup{}
	mainWG.Add(len(addresses))
	for _, addr := range addresses {
		_addr := addr
		go func() {
			defer mainWG.Done()
			testAddress(t, _addr, false)
		}()
	}
	mainWG.Wait()
	dur := time.Since(now)

	log.Printf("TestHandleMultipleRequest for %v addresses finished in %v \n", len(addresses), dur)
}

func TestGetTransactionRequestTime(t *testing.T) {
	log.Println("TestGetTransactionRequestTime start")
	// only 20 transactions in this block, should be good to test time
	addresses := getAddressesForBlock(7000002)
	now := time.Now()
	mainWG := sync.WaitGroup{}
	mainWG.Add(len(addresses))
	for _, addr := range addresses {
		_addr := addr
		go func() {
			defer mainWG.Done()
			// test time
			testAddress(t, _addr, true)
		}()
	}
	mainWG.Wait()
	dur := time.Since(now)

	log.Printf("TestRequestTime for %v addresses finished in %v \n", len(addresses), dur)
}

func TestGetTotalRequestTime(t *testing.T) {
	log.Println("TestGetTotalRequestTime")
	// this address has a lot lot of transactions
	addr := "0x8d12A197cB00D4747a1fe03395095ce2A5CC6819"
	totalUrl := IndexerUrl + "/total"
	url := fmt.Sprintf(totalUrl, addr)
	now := time.Now()
	res, err := getClient().Get(url)
	dur := time.Since(now)
	log.Printf("Get total transaction for %v takes %v \n", addr, dur)
	assert.Nil(t, err)
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	httpResult := httpTypes.EITotalTransaction{}
	err = json.NewDecoder(res.Body).Decode(&httpResult)
	assert.Nil(t, err)
	assert.True(t, httpResult.Total > 10000000)
	log.Printf("Total transaction for %v: %v \n", addr, httpResult.Total)
}

// return number of milliseond
// can't count time if running a lot of http request as the same time because httpClient has a connection pool inside
func testAddress(t *testing.T, addr string, testTime bool) float64 {
	now := time.Now()
	url := fmt.Sprintf(IndexerUrl, addr)
	res, err := getClient().Get(url)
	dur := time.Since(now)
	if err != nil {
		log.Fatalf("Received error %v for address %v url=%v \n", err.Error(), addr, url)
		return 0
	}
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	httpResult := httpTypes.EITransactionsByAccount{}
	err = json.NewDecoder(res.Body).Decode(&httpResult)
	assert.Nil(t, err)
	assert.True(t, len(httpResult.Indexes) > 0)
	total, err := strconv.Atoi(httpResult.Total)
	assert.Nil(t, err)
	assert.True(t, total > 0)
	log.Printf("Address %v has %v transactions, indexer returns in %v \n", addr, httpResult.Total, dur)
	if testTime {
		assert.True(t, dur < 1*time.Second)
	}
	return dur.Seconds()
}

func getAddressesFromBlockchain() []string {
	// ${count} address from block 7000000
	result := []string{}
	for block := StartBlock; block < (BlockCount + StartBlock); block++ {
		addrs := getAddressesForBlock(block)
		result = append(result, addrs...)
	}
	return result
}

func getAddressesForBlock(block int) []string {
	client, _ := ethclient.Dial(EthereumNode)
	ctx := context.Background()
	addrMap := map[string]int{}
	aBlock, _ := client.BlockByNumber(ctx, big.NewInt(int64(block)))
	for index, tx := range aBlock.Transactions() {
		senderAddr, _ := client.TransactionSender(ctx, tx, aBlock.Hash(), uint(index))
		sender := senderAddr.String()
		_, ok := addrMap[sender]
		if !ok {
			addrMap[sender] = 0
		}
		addrMap[sender]++
		to := ""
		if tx.To() != nil {
			to = tx.To().String()
			_, ok = addrMap[to]
			if !ok {
				addrMap[to] = 0
			}
			addrMap[to]++
		}
	}

	result := []string{}
	for key, _ := range addrMap {
		result = append(result, key)
	}
	return result
}
