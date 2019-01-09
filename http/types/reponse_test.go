package types

import (
	"encoding/json"
	"log"
	"math/big"
	"testing"

	coreTypes "github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/stretchr/testify/assert"
)

var index = coreTypes.AddressIndex{
	AddressSequence: coreTypes.AddressSequence{
		Address:  "from1",
		Sequence: 1,
	},
	TxHash:        "0xtx1",
	Value:         big.NewInt(-111),
	Time:          big.NewInt(1546848896),
	BlockNumber:   big.NewInt(2018),
	CoupleAddress: "to1",
}

func TestMarshall(t *testing.T) {
	response := EITransactionsByAccount{
		Total:   10,
		Start:   5,
		Indexes: []coreTypes.AddressIndex{index},
	}
	data, err := json.Marshal(response)
	assert.Nil(t, err)
	assert.Nil(t, err)
	dataStr := string(data)
	log.Printf("%v \n", dataStr)
	expectedStr := `{"numFound":10,"start":5,"data":[{"address":"from1","sequence":1,"tx_hash":"0xtx1","value":-111,"time":1546848896,"blockNumber":2018,"coupleAddress":"to1"}]}`
	assert.Equal(t, expectedStr, dataStr)
}
