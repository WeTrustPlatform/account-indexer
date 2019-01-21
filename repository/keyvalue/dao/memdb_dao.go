package dao

import (
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// MemDbDAO an in-memory dao implementation using memdb package of leveldb
type MemDbDAO struct {
	db *memdb.DB
}

// NewMemDbDAO new memdb dao instance
func NewMemDbDAO(db *memdb.DB) MemDbDAO {
	return MemDbDAO{db: db}
}

// Put implement interface
func (md MemDbDAO) Put(record KeyValue) error {
	err := md.db.Put(record.Key, record.Value)
	return err
}

// BatchPut implement interface
func (md MemDbDAO) BatchPut(records []KeyValue) error {
	for _, item := range records {
		err := md.db.Put(item.Key, item.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// BatchDelete implement interface
func (md MemDbDAO) BatchDelete(keys [][]byte) error {
	for _, key := range keys {
		err := md.DeleteByKey(key)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteByKey implement interface
func (md MemDbDAO) DeleteByKey(key []byte) error {
	err := md.db.Delete(key)
	return err
}

// FindByKeyPrefix implement interface
func (md MemDbDAO) FindByKeyPrefix(prefix []byte, asc bool, rows int, start int) (int, []KeyValue) {
	iter := md.db.NewIterator(util.BytesPrefix(prefix))
	defer iter.Release()
	return findByKeyPrefix(iter, asc, rows, start)
}

// FindByRange find by a range
func (md MemDbDAO) FindByRange(rg *util.Range, asc bool, rows int, start int) (int, []KeyValue) {
	iter := md.db.NewIterator(rg)
	defer iter.Release()
	return findByKeyPrefix(iter, asc, rows, start)
}

// FindByKey implement interface
func (md MemDbDAO) FindByKey(key []byte) (*KeyValue, error) {
	value, err := md.db.Get(key)
	if err != nil {
		return nil, err
	}
	kv := CopyKeyValue(key, value)
	return &kv, nil
}

// GetNFirstRecords implement interface
func (md MemDbDAO) GetNFirstRecords(n int) []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	return getNFirstRecords(iter, n)
}

// GetNLastRecords implement interface
func (md MemDbDAO) GetNLastRecords(n int) []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	return getNLastRecords(iter, n)
}

// GetNFirstPredicate go from first record until predicate evaluation is false
func (md MemDbDAO) GetNFirstPredicate(prep Predicate) []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	return getNFirstPredicate(iter, prep)
}

// GetAllRecords implement interface
func (md MemDbDAO) GetAllRecords() []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	return getAllRecords(iter)
}
