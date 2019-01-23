package dao

import (
	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
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

// Put put a single KeyValue
func (ld LevelDbDAO) Put(record KeyValue) error {
	err := ld.db.Put(record.Key, record.Value, nil)
	return err
}

// BatchPut put an array in batch
func (ld LevelDbDAO) BatchPut(records []KeyValue) error {
	batch := new(leveldb.Batch)
	for _, item := range records {
		batch.Put(item.Key, item.Value)
	}
	err := ld.db.Write(batch, nil)
	return err
}

// BatchDelete delete by key array
func (ld LevelDbDAO) BatchDelete(keys [][]byte) error {
	batch := new(leveldb.Batch)
	for _, key := range keys {
		batch.Delete(key)
	}
	err := ld.db.Write(batch, nil)
	return err
}

// DeleteByKey delete by a key
func (ld LevelDbDAO) DeleteByKey(key []byte) error {
	err := ld.db.Delete(key, nil)
	return err
}

// FindByKeyPrefix find by a key prefix
func (ld LevelDbDAO) FindByKeyPrefix(prefix []byte, asc bool, rows int, start int) (int, []KeyValue) {
	iter := ld.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()
	return findByKeyPrefix(iter, asc, rows, start)
}

// FindByRange find by a range
func (ld LevelDbDAO) FindByRange(rg *util.Range, asc bool, rows int, start int) (int, []KeyValue) {
	iter := ld.db.NewIterator(rg, nil)
	defer iter.Release()
	return findByKeyPrefix(iter, asc, rows, start)
}

func findByKeyPrefix(iter iterator.Iterator, asc bool, rows int, start int) (int, []KeyValue) {
	result := []KeyValue{}
	count := 0
	total := 0

	addToResult := func() {
		if total >= start && count < rows {
			keyValue := CopyKeyValue(iter.Key(), iter.Value())
			result = append(result, keyValue)
			count++
		}
	}

	fn := iter.Next
	if !asc {
		fn = iter.Prev
		hasLast := iter.Last()
		if !hasLast {
			return 0, result
		}
		addToResult()
		total++
	}

	// handle different for asc and desc!!
	for fn() {
		addToResult()
		total++
		// Due to the nature of LevelDB, don't want to loop thru the result if it's a lot
		if total > common.NumMaxTransaction {
			break
		}
	}

	return total, result
}

// FindByKey find by a key
func (ld LevelDbDAO) FindByKey(key []byte) (*KeyValue, error) {
	value, err := ld.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	result := KeyValue{Key: key, Value: value}
	return &result, nil
}

// GetNFirstRecords get n first records
func (ld LevelDbDAO) GetNFirstRecords(n int) []KeyValue {
	iter := ld.db.NewIterator(nil, nil)
	defer iter.Release()
	return getNFirstRecords(iter, n)
}

func getNFirstRecords(iter iterator.Iterator, n int) []KeyValue {
	count := 0
	result := []KeyValue{}
	for count < n && iter.Next() {
		count++
		result = append(result, CopyKeyValue(iter.Key(), iter.Value()))
	}
	return result
}

// GetNLastRecords get n last records
func (ld LevelDbDAO) GetNLastRecords(n int) []KeyValue {
	iter := ld.db.NewIterator(nil, nil)
	defer iter.Release()
	return getNLastRecords(iter, n)
}

func getNLastRecords(iter iterator.Iterator, n int) []KeyValue {
	result := []KeyValue{}
	if !iter.Last() {
		return result
	}

	key := iter.Key()
	value := iter.Value()
	result = append(result, CopyKeyValue(key, value))

	count := 0
	for count < (n-1) && iter.Prev() {
		key := iter.Key()
		value := iter.Value()
		result = append(result, CopyKeyValue(key, value))
		count++
	}
	return result
}

// GetNFirstPredicate go from first record until predicate evaluation is false
func (ld LevelDbDAO) GetNFirstPredicate(prep Predicate) []KeyValue {
	iter := ld.db.NewIterator(nil, nil)
	defer iter.Release()
	return getNFirstPredicate(iter, prep)
}

func getNFirstPredicate(iter iterator.Iterator, prep Predicate) []KeyValue {
	result := []KeyValue{}
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		keyValue := CopyKeyValue(key, value)
		if !prep(keyValue) {
			break
		}
		result = append(result, keyValue)
	}
	return result
}

// GetAllRecords get all records
func (ld LevelDbDAO) GetAllRecords() []KeyValue {
	iter := ld.db.NewIterator(nil, nil)
	defer iter.Release()
	return getAllRecords(iter)
}

func getAllRecords(iter iterator.Iterator) []KeyValue {
	result := []KeyValue{}
	for iter.Next() {
		result = append(result, CopyKeyValue(iter.Key(), iter.Value()))
	}
	return result
}
