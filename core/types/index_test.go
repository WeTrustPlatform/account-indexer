package types

import (
	"encoding/json"
	"log"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// var blockTime = big.NewInt(time.Now().Unix())

var index = AddressIndex{
	AddressSequence: AddressSequence{
		Address:  "from1",
		Sequence: 1,
	},
	TxHash: "0xtx1",
	Value:  big.NewInt(-111),
	Time:   big.NewInt(1546848896),
	// BlockNumber:   big.NewInt(2018),
	CoupleAddress: "to1",
	Status:        true,
}

func TestMarshall(t *testing.T) {
	data, err := json.Marshal(index)
	assert.Nil(t, err)
	dataStr := string(data)
	log.Printf("%v \n", dataStr)
	// expectedJSON := `{"address":"from1","sequence":1,"tx_hash":"0xtx1","value":-111,"time":1546848896,"blockNumber":2018,"coupleAddress":"to1"}`
	expectedJSON := `{"address":"from1","sequence":1,"tx_hash":"0xtx1","value":-111,"time":1546848896,"coupleAddress":"to1","status":true}`
	assert.Equal(t, expectedJSON, dataStr)
	data2, err := json.Marshal(&index)
	assert.Nil(t, err)
	assert.Equal(t, data, data2)
}
