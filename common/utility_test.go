package common

import (
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMarshallTime(t *testing.T) {
	expectedTime := big.NewInt(1546522340)
	testTime(t, expectedTime, expectedTime)
	inTime2 := big.NewInt(1546522340000)
	testTime(t, inTime2, expectedTime)
}

func testTime(t *testing.T, inTime *big.Int, expectedTime *big.Int) {
	byteArr := MarshallTime(inTime)
	if len(byteArr) != 4 {
		t.Error("Byte array should has length 4")
	}
	if UnmarshallTimeToInt(byteArr).Cmp(expectedTime) != 0 {
		t.Error("Marshall/Unmarshall is not correct")
	}
	outTime := UnmarshallTime(byteArr)
	outTimeInt64 := outTime.Unix()
	if outTimeInt64 != int64(expectedTime.Uint64()) {
		t.Error("Unmarshall to time is not correct")
	}
}

func TestStrToUnixTime(t *testing.T) {
	i, err := StrToUnixTimeInt("1405544146")
	assert.Nil(t, err)
	tm := time.Unix(i.Int64(), 0)
	log.Println(tm)
	i, err = StrToUnixTimeInt("140554414")
	assert.Nil(t, err)
	tm = time.Unix(i.Int64(), 0)
	log.Println(tm)
	i, err = StrToUnixTimeInt("")
	assert.NotNil(t, err)
	_, err = StrToUnixTimeInt("100a")
	assert.NotNil(t, err)
}
