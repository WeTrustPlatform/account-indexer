package dao

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// LevelDbDAO a dao implementation using leveldb
type LevelDbDAO struct {
	db *leveldb.DB
}

// NewLevelDbDAO New instance of LevelDbDAO struct
func NewLevelDbDAO(db *leveldb.DB) LevelDbDAO {
	return LevelDbDAO{db: db}
}

func (ld LevelDbDAO) Put(record KeyValue) error {
	err := ld.db.Put(record.key, record.value, nil)
	return err
}

func (ld LevelDbDAO) BatchPut(records []KeyValue) error {
	batch := new(leveldb.Batch)
	for _, item := range records {
		batch.Put(item.key, item.value)
	}
	err := ld.db.Write(batch, nil)
	return err
}

func (ld LevelDbDAO) BatchDelete(keys [][]byte) error {
	batch := new(leveldb.Batch)
	for _, key := range keys {
		batch.Delete(key)
	}
	err := ld.db.Write(batch, nil)
	return err
}

func (ld LevelDbDAO) DeleteByKey(key []byte) error {
	err := ld.db.Delete(key, nil)
	return err
}

func (ld LevelDbDAO) FindByKeyPrefix(prefix []byte) ([]KeyValue, error) {
	iter := ld.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()
	result := []KeyValue{}
	for iter.Next() {
		keyValue := KeyValue{key: iter.Key(), value: iter.Value()}
		result = append(result, keyValue)
	}
	err := iter.Error()
	return result, err
}

func (ld LevelDbDAO) FindByKey(key []byte) (*KeyValue, error) {
	value, err := ld.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	result := KeyValue{key: key, value: value}
	return &result, nil
}

func (ld LevelDbDAO) GetNFirstRecords(n int) []KeyValue {
	iter := ld.db.NewIterator(nil, nil)
	defer iter.Release()
	count := 0
	result := []KeyValue{}
	for iter.Next() && count < n {
		count++
		result = append(result, KeyValue{key: iter.Key(), value: iter.Value()})
	}
	return result
}

func (ld LevelDbDAO) GetNLastRecords(n int) []KeyValue {
	iter := ld.db.NewIterator(nil, nil)
	defer iter.Release()
	count := 0
	result := []KeyValue{}
	hasLast := iter.Last()
	if !hasLast {
		return result
	}
	for iter.Prev() && count < n {
		count++
		result = append(result, KeyValue{key: iter.Key(), value: iter.Value()})
	}
	return result
}

func (ld LevelDbDAO) GetAllRecords() []KeyValue {
	iter := ld.db.NewIterator(nil, nil)
	defer iter.Release()
	result := []KeyValue{}
	for iter.Next() {
		result = append(result, KeyValue{key: iter.Key(), value: iter.Value()})
	}
	return result
}
