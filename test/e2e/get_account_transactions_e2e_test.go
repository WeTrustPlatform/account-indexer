// +build e2e

package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/WeTrustPlatform/account-indexer/common"
	indexerHttp "github.com/WeTrustPlatform/account-indexer/http"
	httpTypes "github.com/WeTrustPlatform/account-indexer/http/types"
	"github.com/stretchr/testify/assert"
)

func TestGetAccountTransactions(t *testing.T) {
	// lupus master address
	account := "0x7C419672d84a53B0a4eFed57656Ba5e4A0379084"
	blockNumbers, eTXHashes := getDataFromEtherScan(t, account)
	indexBlocks(t, blockNumbers)
	iTXHashes := getTXHashesFromIndexer(t, account, len(eTXHashes))
	assert.Equal(t, len(eTXHashes), len(iTXHashes))
	// All Etherscan transaction appears in our indexer
	assert.Equal(t, eTXHashes, iTXHashes)
	// This will print out missing transaction if any
	for _, eTX := range eTXHashes {
		if !common.Contains(iTXHashes, eTX) {
			assert.Fail(t, "The indexer does not contain "+eTX)
		}
	}
}

func indexBlocks(t *testing.T, blockNumbers []string) {
	userName := os.Getenv(indexerHttp.AdminUserName)
	password := os.Getenv(indexerHttp.AdminPassword)
	encoded := base64.StdEncoding.EncodeToString([]byte(userName + ":" + password))
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	for _, block := range blockNumbers {
		url := fmt.Sprintf("http://localhost:3000/admin/blocks/%v", block)
		req, err := http.NewRequest("POST", url, bytes.NewBufferString("{}"))
		assert.Nil(t, err)
		req.Header.Set("Authorization", "Basic "+encoded)
		res, err := httpClient.Do(req)
		assert.Nil(t, err)
		if err == nil {
			assert.Equal(t, 200, res.StatusCode)
			log.Printf("Index block %v successfully \n", block)
		} else {
			log.Fatal("Error indexing block, error=" + err.Error())
		}

		defer res.Body.Close()
	}
}

func getTXHashesFromIndexer(t *testing.T, account string, rows int) []string {
	url := fmt.Sprintf("http://localhost:3000/api/v1/accounts/%v?rows=%v", account, rows)
	t.Logf("Getting index data from %v \n", url)
	res, err := http.Get(url)
	assert.Nil(t, err)
	defer res.Body.Close()
	var httpResult httpTypes.EITransactionsByAccount
	err = json.NewDecoder(res.Body).Decode(&httpResult)
	assert.Nil(t, err)
	assert.Equal(t, rows, len(httpResult.Indexes))
	txHashes := []string{}
	for _, item := range httpResult.Indexes {
		txHashes = append(txHashes, item.TxHash)
	}
	assert.Equal(t, rows, len(txHashes))
	return txHashes
}

// Return list of block numbers and list of transactions
func getDataFromEtherScan(t *testing.T, account string) ([]string, []string) {
	url := fmt.Sprintf("https://etherscan.io/txs?a=%v", account)
	res, err := http.Get(url)
	assert.Nil(t, err)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	assert.Nil(t, err)
	blockNumbers := []string{}
	blockSelector := "div#ContentPlaceHolder1_mainrow > div > div > div > table > tbody > tr > td.hidden-sm > a"
	doc.Find(blockSelector).Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		blockNumbers = append(blockNumbers, text)
	})

	txHashes := []string{}
	txSelector := "div#ContentPlaceHolder1_mainrow > div > div > div > table > tbody > tr > td > span.address-tag > a"
	doc.Find(txSelector).Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		// log.Printf("text=%v", text)
		if len(text) == 66 {
			// transaction, not address
			txHashes = append(txHashes, text)
		}
	})

	return blockNumbers, txHashes
}
