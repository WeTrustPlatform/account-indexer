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
	FindByRange(rg *util.Range, asc bool, rows int, start int) (int, []KeyValue)
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

func NewKeyValue(key []byte, value []byte) KeyValue {
	return KeyValue{Key: key, Value: value}
}

func CopyKeyValue(key []byte, value []byte) KeyValue {
	return KeyValue{Key: copy(key), Value: copy(value)}
}

type Predicate func(KeyValue) bool

func copy(arr []byte) []byte {
	newSlice := make([]byte, len(arr))
	for i, item := range arr {
		newSlice[i] = item
	}
	return newSlice
}
