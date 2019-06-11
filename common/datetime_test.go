package common

import (
	"math/big"
	"strconv"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
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
	log.Info(tm)
	i, err = StrToUnixTimeInt("140554414")
	assert.Nil(t, err)
	tm = time.Unix(i.Int64(), 0)
	log.Info(tm)
	_, err = StrToUnixTimeInt("")
	assert.NotNil(t, err)
	_, err = StrToUnixTimeInt("100a")
	assert.NotNil(t, err)
}

func TestStrToTime(t *testing.T) {
	tm, err := StrToTime("1405544146")
	log.Info(tm)
	assert.Nil(t, err)
	tm, err = StrToTime("2019-01-19T15:04:05+0100")
	log.Info(tm)
	assert.Nil(t, err)
	tm, err = StrToTime("2019-01-19T15:04:05")
	log.Info(tm)
	assert.Nil(t, err)
	tm, err = StrToTime("2019-01-25T07:01:00.691Z")
	log.Info(tm)
	assert.Nil(t, err)
}

func TestPerformance(t *testing.T) {
	// old way
	tm := big.NewInt(1405544146)
	byteArr1 := MarshallTime(tm)

	tm2, _ := StrToTime("2019-01-19T15:04:05+0100")
	byteArr2, _ := tm2.MarshalBinary()
	assert.True(t, len(byteArr1) < len(byteArr2))

	byteArr3 := []byte(strconv.Itoa(1405544146))
	assert.True(t, len(byteArr1) < len(byteArr3))

	byteArr4 := MarshallTime2(tm2)
	assert.Equal(t, len(byteArr1), len(byteArr4))
}
