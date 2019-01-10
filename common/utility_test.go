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
	assert.Equal(t, 4, len(byteArr))
	assert.True(t, UnmarshallTimeToInt(byteArr).Cmp(expectedTime) == 0)
	outTime := UnmarshallTime(byteArr)
	outTimeInt64 := outTime.Unix()
	assert.Equal(t, int64(expectedTime.Uint64()), outTimeInt64)
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
