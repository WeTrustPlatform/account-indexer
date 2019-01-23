package dao

import (
	"github.com/syndtr/goleveldb/leveldb/util"
)

// KeyValueDAO generic DAO interface for the indexer
type KeyValueDAO interface {
	Put(record KeyValue) error
	BatchPut(records []KeyValue) error
	BatchDelete(keys [][]byte) error
	DeleteByKey(key []byte) error
	FindByKeyPrefix(prefix []byte, asc bool, rows int, start int) (int, []KeyValue)
	CountByKeyPrefix(prefix []byte) int
	FindByRange(rg *util.Range, asc bool, rows int, start int) (int, []KeyValue)
	CountByRange(rg *util.Range) int
	FindByKey(key []byte) (*KeyValue, error)
	GetNFirstRecords(n int) []KeyValue
	GetNLastRecords(n int) []KeyValue
	GetNFirstPredicate(pre Predicate) []KeyValue
	GetAllRecords() []KeyValue
}

// KeyValue LevelDB uses key-value struct
type KeyValue struct {
	Key   []byte
	Value []byte
}

// NewKeyValue new KeyValue
func NewKeyValue(key []byte, value []byte) KeyValue {
	return KeyValue{Key: key, Value: value}
}

// CopyKeyValue clone and return enw KeyValue
func CopyKeyValue(key []byte, value []byte) KeyValue {
	return KeyValue{Key: clone(key), Value: clone(value)}
}

// Predicate predicate
type Predicate func(KeyValue) bool

func clone(arr []byte) []byte {
	newSlice := make([]byte, len(arr))
	// for i, item := range arr {
	// 	newSlice[i] = item
	// }
	copy(newSlice, arr)
	return newSlice
}
