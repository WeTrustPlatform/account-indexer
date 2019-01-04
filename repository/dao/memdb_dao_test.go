package dao

import (
	"testing"

	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
)

func TestFindByKeyPrefix(t *testing.T) {
	db := memdb.New(comparer.DefaultComparer, 0)
	dao := NewMemDbDAO(db)
	keyValues := []KeyValue{
		KeyValue{key: []byte("key1"), value: []byte("value1")},
		KeyValue{key: []byte("key2"), value: []byte("value2")},
		KeyValue{key: []byte("strange_key1"), value: []byte("strange_value1")},
	}
	err := dao.BatchPut(keyValues)
	if err != nil {
		t.Error("BatchPut failed with error: " + err.Error())
		return
	}
	prefixFound, err := dao.FindByKeyPrefix([]byte("key"))
	if err != nil {
		t.Error("FindByKeyPrefix failed with error: " + err.Error())
		return
	}
	if len(prefixFound) != 2 {
		t.Error("FindByKeyPrefix failed because len is not correct")
		return
	}
	// TODO: test other functions
}
