package repository

import (
	"log"
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/dao"
	"github.com/WeTrustPlatform/account-indexer/repository/marshal"
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
	addressDAO dao.KeyValueDAO
	blockDAO   dao.KeyValueDAO
	batchDAO   dao.KeyValueDAO
	marshaller marshal.Marshaller
}

// NewLevelDBRepo create an instance of LevelDBRepo
func NewLevelDBRepo(addressDAO dao.KeyValueDAO, blockDAO dao.KeyValueDAO, batchDAO dao.KeyValueDAO) *LevelDBRepo {
	return &LevelDBRepo{
		addressDAO: addressDAO,
		blockDAO:   blockDAO,
		batchDAO:   batchDAO,
		marshaller: marshal.ByteMarshaller{},
	}
}

// Store implements Repository
func (repo *LevelDBRepo) Store(addressIndex []types.AddressIndex, blockIndex types.BlockIndex, isBatch bool) {
	if !isBatch {
		oldBlock, err := repo.blockDAO.FindByKey([]byte(blockIndex.BlockNumber))
		if err == nil && oldBlock != nil {
			reorgAddresses := repo.marshaller.UnmarshallBlockDBValue(oldBlock.Value)
			if reorgAddresses != nil {
				repo.HandleReorg(blockIndex.BlockNumber, reorgAddresses)
			}
		}
	}

	// AddressDB: write in batch
	keyValues := []dao.KeyValue{}
	for _, item := range addressIndex {
		key := repo.marshaller.MarshallAddressKey(item)
		value := repo.marshaller.MarshallAddressValue(item)
		keyValue := dao.NewKeyValue(key, value)
		keyValues = append(keyValues, keyValue)
	}
	err := repo.addressDAO.BatchPut(keyValues)
	if err != nil {
		log.Fatal("Cannot write to address leveldb")
	}

	// BlockDB: write a single record
	if !isBatch {
		key := repo.marshaller.MarshallBlockKey(blockIndex.BlockNumber)
		value := repo.marshaller.MarshallBlockDBValue(blockIndex)
		err = repo.blockDAO.Put(dao.NewKeyValue(key, value))
		if err != nil {
			log.Fatal("Cannot write to block leveldb")
		}
	}
}

// GetTransactionByAddress main thing for this indexer
func (repo *LevelDBRepo) GetTransactionByAddress(address string) []types.AddressIndex {
	prefix := repo.marshaller.MarshallAddressKeyPrefix(address)
	result := []types.AddressIndex{}
	keyValues, _ := repo.addressDAO.FindByKeyPrefix(prefix)
	for _, keyValue := range keyValues {
		value := keyValue.Value
		addressIndex := repo.marshaller.UnmarshallAddressValue(value)
		addressIndex.Address = address
		key := keyValue.Key
		_, blockNumber := repo.marshaller.UnmarshallAddressKey(key)
		addressIndex.BlockNumber = *blockNumber
		result = append(result, addressIndex)
	}

	return result
}

// HandleReorg handle reorg scenario: get block again
func (repo *LevelDBRepo) HandleReorg(blockIndex string, reorgAddresses []types.AddressSequence) {
	keys := [][]byte{}
	for _, address := range reorgAddresses {
		// Block database save address and max sequence as value
		for i := uint8(1); i <= address.Sequence; i++ {
			addressIndexKey := repo.marshaller.MarshallAddressKeyStr(address.Address, blockIndex, i)
			keys = append(keys, addressIndexKey)
		}
	}
	err := repo.addressDAO.BatchDelete(keys)
	if err != nil {
		log.Fatal("Cannot remove old address index")
	}
}

// GetLastNewHeadBlockInDB latest saved block in newHead block DB
func (repo *LevelDBRepo) GetLastNewHeadBlockInDB() *big.Int {
	lastBlocks := repo.blockDAO.GetNLastRecords(1)
	if len(lastBlocks) <= 0 {
		return nil
	}
	key := lastBlocks[0].Key
	blockNumber := repo.marshaller.UnmarshallBlockKey(key)
	return blockNumber
}

// GetFirstNewHeadBlockInDB first saved block in newHead block DB
func (repo *LevelDBRepo) GetFirstNewHeadBlockInDB() *big.Int {
	lastBlocks := repo.blockDAO.GetNFirstRecords(1)
	if len(lastBlocks) <= 0 {
		return nil
	}
	key := lastBlocks[0].Key
	blockNumber := repo.marshaller.UnmarshallBlockKey(key)
	return blockNumber
}

// GetAllBatchStatuses get all batches
func (repo *LevelDBRepo) GetAllBatchStatuses() []types.BatchStatus {
	keyValues := repo.batchDAO.GetAllRecords()
	batches := []types.BatchStatus{}
	for _, keyValue := range keyValues {
		key := keyValue.Key
		value := keyValue.Value
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
	repo.batchDAO.Put(dao.NewKeyValue(key, value))
}

func (repo *LevelDBRepo) ReplaceBatch(from *big.Int, newTo *big.Int) {
	fromByteArr := repo.marshaller.MarshallBatchKeyFrom(from)
	keyValues, err := repo.batchDAO.FindByKeyPrefix(fromByteArr)
	if err != nil || len(keyValues) <= 0 {
		return
	}
	keyValue := keyValues[0]
	key := keyValue.Key
	value := keyValue.Value
	batch := repo.getBatchStatus(key, value)
	repo.replaceBatch(batch, newTo)
}

func (repo *LevelDBRepo) replaceBatch(batch types.BatchStatus, newTo *big.Int) {
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To)
	repo.batchDAO.DeleteByKey(key)
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
	keyValues := repo.blockDAO.GetNLastRecords(5)
	result := []types.BlockIndex{}
	for _, keyValue := range keyValues {
		key := keyValue.Key
		value := keyValue.Value
		blockNumber := repo.marshaller.UnmarshallBlockKey(key)
		reorgAddresses := repo.marshaller.UnmarshallBlockDBValue(value)
		result = append(result, types.BlockIndex{
			BlockNumber: blockNumber.String(),
			Addresses:   reorgAddresses,
		})
	}
	return result
}
