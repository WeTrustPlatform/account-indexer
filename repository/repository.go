package repository

import (
	"log"
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/marshal"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Repository to store index data
type Repository interface {
	Store(indexData []types.AddressIndex, blockIndex types.BlockIndex, isBatch bool)
	GetTransactionByAddress(address string) []types.AddressIndex
	HandleReorg(blockIndex string, reorgAddresses []types.AddressSequence)
	GetLastNewHeadBlockInDB() *big.Int
	GetFirstNewHeadBlockInDB() *big.Int
	GetAllBatchStatuses() []types.BatchStatus
	UpdateBatch(batch types.BatchStatus)
	ReplaceBatch(from *big.Int, newTo *big.Int)
	GetLastFiveBlocks() []types.BlockIndex
}

// LevelDBRepo implementation of Repository
type LevelDBRepo struct {
	addressDB  *leveldb.DB
	blockDB    *leveldb.DB
	batchDB    *leveldb.DB
	marshaller marshal.Marshaller
}

// NewLevelDBRepo create an instance of LevelDBRepo
func NewLevelDBRepo(addressDB *leveldb.DB, blockDB *leveldb.DB, batchDB *leveldb.DB) *LevelDBRepo {
	return &LevelDBRepo{
		addressDB:  addressDB,
		blockDB:    blockDB,
		batchDB:    batchDB,
		marshaller: marshal.ByteMarshaller{},
	}
}

// Store implements Repository
func (repo *LevelDBRepo) Store(addressIndex []types.AddressIndex, blockIndex types.BlockIndex, isBatch bool) {
	batch := new(leveldb.Batch)
	if !isBatch {
		reorgAddressesByteArr, _ := repo.blockDB.Get([]byte(blockIndex.BlockNumber), nil)
		if reorgAddressesByteArr != nil {
			reorgAddresses := repo.marshaller.UnmarshallBlockDBValue(reorgAddressesByteArr)
			if reorgAddresses != nil {
				repo.HandleReorg(blockIndex.BlockNumber, reorgAddresses)
			}
		}
	}

	// AddressDB: write in batch
	for _, item := range addressIndex {
		key := repo.marshaller.MarshallAddressKey(item)
		value := repo.marshaller.MarshallAddressValue(item)
		batch.Put(key, value)
	}
	err := repo.addressDB.Write(batch, nil)
	if err != nil {
		log.Fatal("Cannot write to address leveldb")
	}

	// BlockDB: write a single record
	if !isBatch {
		err = repo.blockDB.Put(repo.marshaller.MarshallBlockKey(blockIndex.BlockNumber), repo.marshaller.MarshallBlockDBValue(blockIndex), nil)
		if err != nil {
			log.Fatal("Cannot write to block leveldb")
		}
	}
}

// GetTransactionByAddress main thing for this indexer
func (repo *LevelDBRepo) GetTransactionByAddress(address string) []types.AddressIndex {
	result := []types.AddressIndex{}
	prefix := repo.marshaller.MarshallAddressKeyPrefix(address)
	iter := repo.addressDB.NewIterator(util.BytesPrefix(prefix), nil)
	for iter.Next() {
		value := iter.Value()
		addressIndex := repo.marshaller.UnmarshallAddressValue(value)
		addressIndex.Address = address
		key := iter.Key()
		_, blockNumber := repo.marshaller.UnmarshallAddressKey(key)
		addressIndex.BlockNumber = *blockNumber
		result = append(result, addressIndex)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Fatal("Cannot get address info from address DB")
	}
	return result
}

// HandleReorg handle reorg scenario: get block again
func (repo *LevelDBRepo) HandleReorg(blockIndex string, reorgAddresses []types.AddressSequence) {
	batch := new(leveldb.Batch)
	for _, address := range reorgAddresses {
		// Block database save address and max sequence as value
		for i := uint8(1); i <= address.Sequence; i++ {
			addressIndexKey := repo.marshaller.MarshallAddressKeyStr(address.Address, blockIndex, i)
			batch.Delete([]byte(addressIndexKey))
		}
	}
	err := repo.addressDB.Write(batch, nil)
	if err != nil {
		log.Fatal("Cannot remove old address index")
	}
}

// GetLastNewHeadBlockInDB latest saved block in newHead block DB
func (repo *LevelDBRepo) GetLastNewHeadBlockInDB() *big.Int {
	iter := repo.blockDB.NewIterator(nil, nil)
	defer iter.Release()
	hasLast := iter.Last()
	if !hasLast {
		return nil
	}
	key := iter.Key()
	blockNumber := repo.marshaller.UnmarshallBlockKey(key)
	return blockNumber
}

// GetFirstNewHeadBlockInDB first saved block in newHead block DB
func (repo *LevelDBRepo) GetFirstNewHeadBlockInDB() *big.Int {
	iter := repo.blockDB.NewIterator(nil, nil)
	defer iter.Release()
	hasFirst := iter.First()
	if !hasFirst {
		return nil
	}
	key := iter.Key()
	blockNumber := repo.marshaller.UnmarshallBlockKey(key)
	return blockNumber
}

// GetAllBatchStatuses get all batches
func (repo *LevelDBRepo) GetAllBatchStatuses() []types.BatchStatus {
	iter := repo.batchDB.NewIterator(nil, nil)
	defer iter.Release()
	batches := []types.BatchStatus{}
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		batch1 := repo.marshaller.UnmarshallBatchKey(key)
		batch2 := repo.marshaller.UnmarshallBatchValue(value)
		batch := types.BatchStatus{
			From:      batch1.From,
			To:        batch1.To,
			UpdatedAt: batch2.UpdatedAt,
			Current:   batch2.Current,
		}
		batches = append(batches, batch)
	}
	return batches
}

// UpdateBatch update a batch
func (repo *LevelDBRepo) UpdateBatch(batch types.BatchStatus) {
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To)
	value := repo.marshaller.MarshallBatchValue(batch.UpdatedAt, batch.Current)
	repo.batchDB.Put(key, value, nil)
}

func (repo *LevelDBRepo) ReplaceBatch(from *big.Int, newTo *big.Int) {
	iter := repo.batchDB.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		batch := repo.getBatchStatus(key, value)
		if batch.From.Cmp(from) == 0 {
			repo.replaceBatch(batch, newTo)
			break
		}
	}
}

func (repo *LevelDBRepo) replaceBatch(batch types.BatchStatus, newTo *big.Int) {
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To)
	repo.batchDB.Delete(key, nil)
	batch.To = newTo
	repo.UpdateBatch(batch)
}

func (repo *LevelDBRepo) getBatchStatus(key []byte, value []byte) types.BatchStatus {
	batch := repo.marshaller.UnmarshallBatchKey(key)
	batchValue := repo.marshaller.UnmarshallBatchValue(value)
	batch.Current = batchValue.Current
	batch.UpdatedAt = batchValue.UpdatedAt
	return batch
}

// GetLastFiveBlocks this is mainly for the server to show the status
func (repo *LevelDBRepo) GetLastFiveBlocks() []types.BlockIndex {
	iter := repo.blockDB.NewIterator(nil, nil)
	defer iter.Release()
	iter.Last()
	result := []types.BlockIndex{}
	count := 0
	for iter.Prev() && count < 5 {
		count++
		key := iter.Key()
		value := iter.Value()
		blockNumber := repo.marshaller.UnmarshallBlockKey(key)
		reorgAddresses := repo.marshaller.UnmarshallBlockDBValue(value)
		result = append(result, types.BlockIndex{
			BlockNumber: blockNumber.String(),
			Addresses:   reorgAddresses,
		})
	}
	return result
}
