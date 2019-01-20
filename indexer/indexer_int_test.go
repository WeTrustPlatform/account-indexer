// // +build int

package indexer

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/fetcher"
	"github.com/WeTrustPlatform/account-indexer/service"
	"github.com/stretchr/testify/assert"
)

func TestContractCreation(t *testing.T) {
	// Setup
	ipcs := []string{"wss://mainnet.kivutar.me:8546/2KT179di"}
	service.GetIpcManager().SetIPC(ipcs)
	fetcher, err := fetcher.NewChainFetch()
	assert.Nil(t, err)
	blockNumber := big.NewInt(6808718)
	// Run Test
	blockDetail, err := fetcher.FetchABlock(blockNumber)
	assert.Nil(t, err)
	// log.Println(blockDetail)
	idx := newTestIndexer()
	isBatch := true
	idx.processBlock(blockDetail, isBatch)
	// Confirm contract created tx
	contract := "0x4a6ead96974679957a17d2f9c7835a3da7ddf91d"
	total, addressIndexes := idx.IndexRepo.GetTransactionByAddress(contract, 10, 0, time.Time{}, time.Time{})
	assert.Equal(t, 1, total)
	tx := addressIndexes[0].TxHash
	assert.True(t, strings.EqualFold("0x61278dd960415eadf11cfe17a6c38397af658e77bbdd367db70e19ee3a193bdd", tx))
}
