package common

import (
	"math/big"
	"strconv"
	"time"
)

// MarshallTime marshall unix time to byte array
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

// MarshallTime2 time to byte array
func MarshallTime2(tm time.Time) []byte {
	timeBigInt := new(big.Int)
	timeBigInt.SetInt64(tm.Unix())
	return timeBigInt.Bytes()
}

// UnmarshallTimeToInt unmarshall a time in byte array to unix
func UnmarshallTimeToInt(value []byte) *big.Int {
	result := new(big.Int)
	result.SetBytes(value)
	return result
}

// UnmarshallTime unmarshall a time in byte array to time
func UnmarshallTime(value []byte) time.Time {
	timeBigInt := UnmarshallTimeToInt(value)
	return UnmarshallIntToTime(timeBigInt)
}

// UnmarshallIntToTime unmarshall a unix time to time
func UnmarshallIntToTime(value *big.Int) time.Time {
	outTime := time.Unix(int64(value.Uint64()), 0)
	return outTime
}

// StrToUnixTimeInt unmarshall a unix time in string to unix time in big int
func StrToUnixTimeInt(str string) (*big.Int, error) {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, err
	}
	return big.NewInt(i), nil
}

// StrToTime supports converting unix and iso8601 format to a time
func StrToTime(str string) (time.Time, error) {
	// unix
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		tm := time.Unix(i, 0)
		return tm, nil
	}
	// ISO 8601 format
	tm, err := time.Parse("2006-01-02T15:04:05-0700", str)
	if err == nil {
		return tm, nil
	}
	// ISO 8601 without zone
	tm, err = time.Parse("2006-01-02T15:04:05", str)
	return tm, err
}
