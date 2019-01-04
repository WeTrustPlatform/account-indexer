package dao

import (
	"github.com/syndtr/goleveldb/leveldb/memdb"
)

// MemDbDAO an in-memory dao implementation using memdb package of leveldb
type MemDbDAO struct {
	db *memdb.DB
}

func NewMemDbDAO(db *memdb.DB) MemDbDAO {
	return MemDbDAO{db: db}
}

func (md MemDbDAO) Put(record KeyValue) error {
	err := md.db.Put(record.Key, record.Value)
	return err
}

func (md MemDbDAO) BatchPut(records []KeyValue) error {
	for _, item := range records {
		err := md.db.Put(item.Key, item.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (md MemDbDAO) BatchDelete(keys [][]byte) error {
	for _, key := range keys {
		err := md.DeleteByKey(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func (md MemDbDAO) DeleteByKey(key []byte) error {
	err := md.db.Delete(key)
	return err
}

func (md MemDbDAO) FindByKeyPrefix(prefix []byte) ([]KeyValue, error) {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	result := []KeyValue{}
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		if startsWith(key, prefix) {
			result = append(result, CopyKeyValue(key, value))
		}
	}
	return result, nil
}

func startsWith(parent []byte, child []byte) bool {
	if len(parent) < len(child) {
		return false
	}
	for index, value := range parent {
		if index < len(child) {
			if child[index] != value {
				return false
			}
		} else {
			break
		}
	}
	return true
}

func (md MemDbDAO) FindByKey(key []byte) (*KeyValue, error) {
	value, err := md.db.Get(key)
	if err != nil {
		return nil, err
	}
	return &KeyValue{Key: key, Value: value}, nil
}

func (md MemDbDAO) GetNFirstRecords(n int) []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	result := []KeyValue{}
	count := 0
	for iter.Next() && count < n {
		key := iter.Key()
		value := iter.Value()
		result = append(result, CopyKeyValue(key, value))
		count++
	}
	return result
}

func (md MemDbDAO) GetNLastRecords(n int) []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	result := []KeyValue{}
	if !iter.Last() {
		return result
	}
	count := 0
	for iter.Prev() && count < n {
		key := iter.Key()
		value := iter.Value()
		result = append(result, CopyKeyValue(key, value))
		count++
	}
	return result
}

func (md MemDbDAO) GetAllRecords() []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	result := []KeyValue{}
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		result = append(result, CopyKeyValue(key, value))
	}
	return result
}
