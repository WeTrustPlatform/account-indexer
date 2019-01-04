package common

import (
	"math/big"
	"time"
)

func MarshallTime(t *big.Int) []byte {
	timeStr := t.String()
	// Unix time is 10 in length
	if len(timeStr) > 10 {
		timeStr = timeStr[:10]
	}
	timeBigInt := new(big.Int)
	timeBigInt.SetString(timeStr, 10)
	return timeBigInt.Bytes()
}

func UnmarshallTimeToInt(value []byte) *big.Int {
	result := new(big.Int)
	result.SetBytes(value)
	return result
}

func UnmarshallTime(value []byte) time.Time {
	timeBigInt := UnmarshallTimeToInt(value)
	return UnmarshallIntToTime(timeBigInt)
}

func UnmarshallIntToTime(value *big.Int) time.Time {
	outTime := time.Unix(int64(value.Uint64()), 0)
	return outTime
}
