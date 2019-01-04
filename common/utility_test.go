package common

import (
	"math/big"
	"testing"
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
