// +build e2e

package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	indexerHttp "github.com/WeTrustPlatform/account-indexer/http"
	httpTypes "github.com/WeTrustPlatform/account-indexer/http/types"
	"github.com/stretchr/testify/assert"
)

func TestConvertEther(t *testing.T) {
	value1FromEtherScan := "0.16310815 Ether"
	assert.Equal(t, "163108150000000000", esValueToString(value1FromEtherScan))
	value2FromEtherScan := "9.56064602 Ether"
	assert.Equal(t, "9560646020000000000", esValueToString(value2FromEtherScan))
	value3FromEtherScan := "1.0182142 Ether"
	assert.Equal(t, "1018214200000000000", esValueToString(value3FromEtherScan))
}

// Get data from etherscan.com and compare to our indexer
func TestGetAccountTransactions(t *testing.T) {
	// lupus master address
	account := "0x7C419672d84a53B0a4eFed57656Ba5e4A0379084"
	blockNumbers, esData := getDataFromEtherScan(t, account)
	indexBlocks(t, blockNumbers)
	indexData := getDataFromIndexer(t, account, len(esData))
	assert.Equal(t, len(esData), len(indexData))
	// This will print out missing data if any
	for i, esTx := range esData {
		tx := esTx.TxHash
		addr := esTx.CoupleAddress
		value := esTx.Value
		assert.Equal(t, tx, indexData[i].TxHash)
		assert.Equal(t, addr, indexData[i].CoupleAddress)
		assert.Equal(t, value, indexData[i].Value)
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

func getDataFromIndexer(t *testing.T, account string, rows int) []httpTypes.EIAddress {
	url := fmt.Sprintf("http://localhost:3000/api/v1/accounts/%v?rows=%v", account, rows)
	t.Logf("Getting index data from %v \n", url)
	res, err := http.Get(url)
	assert.Nil(t, err)
	defer res.Body.Close()
	var httpResult httpTypes.EITransactionsByAccount
	err = json.NewDecoder(res.Body).Decode(&httpResult)
	assert.Nil(t, err)
	assert.Equal(t, rows, len(httpResult.Indexes))
	return httpResult.Indexes
}

// Return list of block numbers and list of AddressIndex
func getDataFromEtherScan(t *testing.T, account string) ([]string, []httpTypes.EIAddress) {
	url := fmt.Sprintf("https://etherscan.io/txs?a=%v", account)
	log.Printf("Getting data from %v \n", url)
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
	txLines := []httpTypes.EIAddress{}
	trSelector := "div#ContentPlaceHolder1_mainrow > div > div > div > table > tbody > tr"
	doc.Find(trSelector).Each(func(i int, s *goquery.Selection) {
		// line by line
		blockNumber := s.Find("td.hidden-sm > a").Text()
		blockNumbers = append(blockNumbers, blockNumber)
		txOrAddr := s.Find("td > span.address-tag > a")
		tx := txOrAddr.First().Text()
		addrNode := txOrAddr.Last()
		addrHref, ok := addrNode.Attr("href")
		assert.True(t, ok)
		addrHrefBA := []byte(addrHref)
		addr := string(addrHrefBA[len("/address/"):])

		valueNode := s.Find("td").Last().Prev()
		valueStr := esValueToString(valueNode.Text())
		value := new(big.Int)
		value.SetString(valueStr, 10)

		txLines = append(txLines, httpTypes.EIAddress{
			TxHash:        tx,
			CoupleAddress: addr,
			Value:         value,
		})
	})

	log.Printf("Done getting data, number of transaction: %v \n", len(txLines))

	return blockNumbers, txLines
}

func esValueToString(str string) string {
	valueBA := []byte(str)
	valueStr := string(valueBA[:len(valueBA)-len(" Ether")])
	f, _ := strconv.ParseFloat(valueStr, 64)
	bf := big.NewFloat(f)
	bf2 := new(big.Float).Mul(bf, big.NewFloat(1000000000000000000))
	f64Val, _ := bf2.Float64()
	result := strconv.FormatFloat(f64Val, 'f', 0, 64)
	// Something wrong with the above conversion, use this as a work around
	result = string([]byte(result)[:len(result)-5]) + "00000"
	return result
}
