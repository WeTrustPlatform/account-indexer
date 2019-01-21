package types

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"testing"
	"time"

	coreTypes "github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/stretchr/testify/assert"
)

var index = coreTypes.AddressIndex{
	AddressSequence: coreTypes.AddressSequence{
		Address:  "from1",
		Sequence: 1,
	},
	TxHash: "0xtx1",
	Value:  big.NewInt(-111),
	Time:   big.NewInt(1546848896),
	// BlockNumber:   big.NewInt(2018),
	CoupleAddress: "to1",
}

func TestMarshall(t *testing.T) {
	idx := AddressToEIAddress(index)
	idx.Data = []byte{1, 2}
	response := EITransactionsByAccount{
		Total:   10,
		Start:   5,
		Indexes: []EIAddress{idx},
	}
	data, err := json.Marshal(response)
	assert.Nil(t, err)
	assert.Nil(t, err)
	dataStr := string(data)
	log.Printf("%v \n", dataStr)
	tm := time.Now()
	tmF := tm.Format(time.RFC3339)
	var timeExt string
	if strings.HasSuffix(tmF, "Z") {
		timeExt = "Z"
	} else {
		byteArr := []byte(tmF)
		extBA := byteArr[len(byteArr)-6:]
		timeExt = string(extBA)
	}
	expectedStr := fmt.Sprintf(`{"numFound":10,"start":5,"data":[{"address":"from1","txHash":"0xtx1","value":-111,"time":"2019-01-07T15:14:56%v","coupleAddress":"to1","data":"AQI=","gas":0,"gasPrice":null}]}`, timeExt)
	assert.Equal(t, expectedStr, dataStr)
}
