package dao

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/memdb"
)

var keyValues = []KeyValue{
	KeyValue{Key: []byte("key1"), Value: []byte("value1")},
	KeyValue{Key: []byte("key2"), Value: []byte("value2")},
	KeyValue{Key: []byte("strange_key1"), Value: []byte("strange_value1")},
}

type MemDbDAOTestSuite struct {
	suite.Suite
	dao MemDbDAO
}

func TestMemDbDAO(t *testing.T) {
	suite.Run(t, new(MemDbDAOTestSuite))
}

func (suite *MemDbDAOTestSuite) SetupTest() {
	db := memdb.New(comparer.DefaultComparer, 0)
	suite.dao = NewMemDbDAO(db)
	err := suite.dao.BatchPut(keyValues)
	assert.Nil(suite.T(), err)
}

func (suite *MemDbDAOTestSuite) TestFindByKeyPrefix() {
	total, prefixFound := suite.dao.FindByKeyPrefix([]byte("key"), true, 10, 0)
	assert.Equal(suite.T(), 2, total)
	assert.Equal(suite.T(), 2, len(prefixFound), "Found items by prefix should be 2")
	assert.True(suite.T(), reflect.DeepEqual(keyValues[0], prefixFound[0]))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[1], prefixFound[1]))

	total, prefixFound = suite.dao.FindByKeyPrefix([]byte("key"), true, 1, 0)
	assert.Equal(suite.T(), 2, total)
	assert.Equal(suite.T(), 1, len(prefixFound), "Found items by prefix should be 1")
	assert.True(suite.T(), reflect.DeepEqual(keyValues[0], prefixFound[0]))

	total, prefixFound = suite.dao.FindByKeyPrefix([]byte("key"), false, 10, 0)
	assert.Equal(suite.T(), 2, total)
	assert.Equal(suite.T(), 2, len(prefixFound), "Found items by prefix should be 2")
	assert.True(suite.T(), reflect.DeepEqual(keyValues[1], prefixFound[0]))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[0], prefixFound[1]))
}

func (suite *MemDbDAOTestSuite) TestGetNFirstPredicate() {
	// 1st test
	pre := Predicate(func(KeyValue) bool {
		return true
	})
	foundKVs := suite.dao.GetNFirstPredicate(pre)
	assert.Equal(suite.T(), 3, len(foundKVs))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[0], foundKVs[0]))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[1], foundKVs[1]))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[2], foundKVs[2]))
	// 2nd test
	pre = Predicate(func(kv KeyValue) bool {
		str := string(kv.Key)
		return strings.HasPrefix(str, "key")
	})
	foundKVs = suite.dao.GetNFirstPredicate(pre)
	assert.Equal(suite.T(), 2, len(foundKVs))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[0], foundKVs[0]))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[1], foundKVs[1]))
}

func (suite *MemDbDAOTestSuite) TestBatchDelete() {
	key1 := []byte("key1")
	key2 := []byte("key2")
	keys := [][]byte{key1, key2}
	err := suite.dao.BatchDelete(keys)
	assert.Nil(suite.T(), err)
	all := suite.dao.GetAllRecords()
	assert.Equal(suite.T(), 1, len(all), "After BatchDelete, it should has 1 item")
	assert.True(suite.T(), reflect.DeepEqual(keyValues[2], all[0]))
}

func (suite *MemDbDAOTestSuite) TestFindByKey() {
	key := []byte("key1")
	kv, err := suite.dao.FindByKey(key)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), keyValues[0].Key, kv.Key)
	assert.Equal(suite.T(), keyValues[0].Value, kv.Value)
}

func (suite *MemDbDAOTestSuite) TestGetNFirstRecords() {
	result := suite.dao.GetNFirstRecords(2)
	assert.Equal(suite.T(), 2, len(result), "Found items by prGetNFirstRecordsefix should be 2")
	assert.True(suite.T(), reflect.DeepEqual(keyValues[0], result[0]))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[1], result[1]))
}

func (suite *MemDbDAOTestSuite) TestGetNLastRecords() {
	result := suite.dao.GetNLastRecords(2)
	assert.Equal(suite.T(), 2, len(result), "Found items by GetNLastRecords should be 2")
	assert.True(suite.T(), reflect.DeepEqual(keyValues[2], result[0]))
	assert.True(suite.T(), reflect.DeepEqual(keyValues[1], result[1]))
}
