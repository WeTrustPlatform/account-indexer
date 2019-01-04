package dao

// IndexerDAO generic DAO interface for the indexer
type KeyValueDAO interface {
	Put(record KeyValue) error
	BatchPut(records []KeyValue) error
	BatchDelete(keys [][]byte) error
	DeleteByKey(key []byte) error
	FindByKeyPrefix(prefix []byte) ([]KeyValue, error)
	FindByKey(key []byte) (*KeyValue, error)
	GetNFirstRecords(n int) []KeyValue
	GetNLastRecords(n int) []KeyValue
	GetAllRecords() []KeyValue
}

// KeyValue LevelDB uses key-value struct
type KeyValue struct {
	key   []byte
	value []byte
}
