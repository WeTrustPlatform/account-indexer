package dao

import (
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
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

func (md MemDbDAO) FindByKey(key []byte) (*KeyValue, error) {
	value, err := md.db.Get(key)
	if err != nil {
		return nil, err
	}
	kv := CopyKeyValue(key, value)
	return &kv, nil
}

func (md MemDbDAO) GetNFirstRecords(n int) []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	return getNFirstRecords(iter, n)
}

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

func (md MemDbDAO) GetAllRecords() []KeyValue {
	iter := md.db.NewIterator(nil)
	defer iter.Release()
	return getAllRecords(iter)
}
