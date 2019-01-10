package repository

import (
	"errors"
	"log"
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/dao"
	"github.com/WeTrustPlatform/account-indexer/repository/marshal"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Repository to store index data
type Repository interface {
	Store(indexData []*types.AddressIndex, blockIndex *types.BlockIndex, isBatch bool) error
	GetTransactionByAddress(address string, rows int, start int, fromTime *big.Int, toTime *big.Int) (int, []types.AddressIndex)
	GetLastNewHeadBlockInDB() *big.Int
	GetFirstNewHeadBlockInDB() *big.Int
	GetAllBatchStatuses() []types.BatchStatus
	UpdateBatch(batch types.BatchStatus) error
	ReplaceBatch(from *big.Int, newTo *big.Int) error
	GetBlocks(blockNumber string, rows int, start int) (int, []types.BlockIndex)
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
func (repo *LevelDBRepo) Store(addressIndex []*types.AddressIndex, blockIndex *types.BlockIndex, isBatch bool) error {
	if !isBatch {
		oldBlock, err := repo.blockDAO.FindByKey([]byte(blockIndex.BlockNumber))
		if err == nil && oldBlock != nil {
			blockIndex := repo.marshaller.UnmarshallBlockValue(oldBlock.Value)
			if blockIndex.Addresses != nil && len(blockIndex.Addresses) > 0 {
				blockTime := blockIndex.Time
				reorgAddresses := blockIndex.Addresses
				err = repo.HandleReorg(blockTime, reorgAddresses)
				if err != nil {
					log.Println("Cannot handle reorg, err=" + err.Error())
					return err
				}
			}
		}
	}

	// AddressDB: write in batch
	err := repo.SaveAddressIndex(addressIndex)
	if err != nil {
		log.Println("Cannot save address index, err=" + err.Error())
		return err
	}

	// BlockDB: write a single record
	if !isBatch {
		err = repo.SaveBlockIndex(blockIndex)
		if err != nil {
			log.Println("Cannot save block index, err=" + err.Error())
		}
	}
	return err
}

// SaveAddressIndex save to address db
func (repo *LevelDBRepo) SaveAddressIndex(addressIndex []*types.AddressIndex) error {
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
	return err
}

// SaveBlockIndex save to block db
func (repo *LevelDBRepo) SaveBlockIndex(blockIndex *types.BlockIndex) error {
	key := repo.marshaller.MarshallBlockKey(blockIndex.BlockNumber)
	value := repo.marshaller.MarshallBlockValue(blockIndex)
	err := repo.blockDAO.Put(dao.NewKeyValue(key, value))
	if err != nil {
		log.Fatal("Cannot write to block leveldb")
	}
	return err
}

// GetTransactionByAddress main thing for this indexer
func (repo *LevelDBRepo) GetTransactionByAddress(address string, rows int, start int, fromTime *big.Int, toTime *big.Int) (int, []types.AddressIndex) {
	convertKeyValuesToAddressIndexes := func(keyValues []dao.KeyValue) []types.AddressIndex {
		result := []types.AddressIndex{}
		for _, keyValue := range keyValues {
			addressIndex := repo.keyValueToAddressIndex(keyValue)
			result = append(result, addressIndex)
		}
		return result
	}

	if fromTime != nil && toTime != nil {
		// assuming fromTime and toTime is good
		// make toTime inclusive
		fromPrefix := repo.marshaller.MarshallAddressKeyPrefix2(address, fromTime)
		newToTime := new(big.Int)
		newToTime = newToTime.Add(toTime, big.NewInt(1))
		toPrefix := repo.marshaller.MarshallAddressKeyPrefix2(address, newToTime)
		rg := &util.Range{Start: fromPrefix, Limit: toPrefix}
		asc := true
		total, keyValues := repo.addressDAO.FindByRange(rg, asc, rows, start)
		addressIndexes := convertKeyValuesToAddressIndexes(keyValues)
		return total, addressIndexes
	}
	// Search by address as LevelDB prefix
	prefix := repo.marshaller.MarshallAddressKeyPrefix(address)
	// bad address
	if len(prefix) == 0 {
		return 0, []types.AddressIndex{}
	}
	asc := false
	total, keyValues := repo.addressDAO.FindByKeyPrefix(prefix, asc, rows, start)
	addressIndexes := convertKeyValuesToAddressIndexes(keyValues)

	return total, addressIndexes
}

func (repo *LevelDBRepo) keyValueToAddressIndex(keyValue dao.KeyValue) types.AddressIndex {
	value := keyValue.Value
	addressIndex := repo.marshaller.UnmarshallAddressValue(value)
	key := keyValue.Key
	address, time := repo.marshaller.UnmarshallAddressKey(key)
	addressIndex.Address = address
	addressIndex.Time = time
	return addressIndex
}

// HandleReorg handle reorg scenario: get block again
func (repo *LevelDBRepo) HandleReorg(blockTime *big.Int, reorgAddresses []types.AddressSequence) error {
	keys := [][]byte{}
	for _, address := range reorgAddresses {
		// Block database save address and max sequence as value
		for i := uint8(1); i <= address.Sequence; i++ {
			addressIndexKey := repo.marshaller.MarshallAddressKeyStr(address.Address, blockTime, i)
			keys = append(keys, addressIndexKey)
		}
	}
	err := repo.addressDAO.BatchDelete(keys)
	return err
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
		batch := repo.keyValueToBatchStatus(keyValue)
		batches = append(batches, batch)
	}
	return batches
}

func (repo *LevelDBRepo) keyValueToBatchStatus(keyValue dao.KeyValue) types.BatchStatus {
	key := keyValue.Key
	value := keyValue.Value
	batch1 := repo.marshaller.UnmarshallBatchKey(key)
	batch2 := repo.marshaller.UnmarshallBatchValue(value)
	batch := types.BatchStatus{
		From:      batch1.From,
		To:        batch1.To,
		UpdatedAt: batch2.UpdatedAt,
		CreatedAt: batch1.CreatedAt,
		Current:   batch2.Current,
	}
	return batch
}

// UpdateBatch update a batch
func (repo *LevelDBRepo) UpdateBatch(batch types.BatchStatus) error {
	if batch.From == nil || batch.To == nil || batch.CreatedAt == nil {
		return errors.New("Batch is not valid, value:" + batch.String())
	}
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To, batch.CreatedAt)
	value := repo.marshaller.MarshallBatchValue(batch.UpdatedAt, batch.Current)
	return repo.batchDAO.Put(dao.NewKeyValue(key, value))
}

// ReplaceBatch replace a batch with new "to"
func (repo *LevelDBRepo) ReplaceBatch(from *big.Int, newTo *big.Int) error {
	fromByteArr := repo.marshaller.MarshallBatchKeyFrom(from)
	asc := true
	_, keyValues := repo.batchDAO.FindByKeyPrefix(fromByteArr, asc, 0, 0)
	if len(keyValues) <= 0 {
		return nil
	}
	keyValue := keyValues[0]
	key := keyValue.Key
	value := keyValue.Value
	batch := repo.getBatchStatus(key, value)
	return repo.replaceBatch(batch, newTo)
}

func (repo *LevelDBRepo) replaceBatch(batch types.BatchStatus, newTo *big.Int) error {
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To, batch.CreatedAt)
	repo.batchDAO.DeleteByKey(key)
	batch.To = newTo
	return repo.UpdateBatch(batch)
}

func (repo *LevelDBRepo) getBatchStatus(key []byte, value []byte) types.BatchStatus {
	batch := repo.marshaller.UnmarshallBatchKey(key)
	batchValue := repo.marshaller.UnmarshallBatchValue(value)
	batch.Current = batchValue.Current
	batch.UpdatedAt = batchValue.UpdatedAt
	return batch
}

// GetBlocks by blockNumber. blockNumber = blank => latest block
func (repo *LevelDBRepo) GetBlocks(blockNumber string, rows int, start int) (int, []types.BlockIndex) {
	result := []types.BlockIndex{}
	makeBlockIndex := func(keyValue *dao.KeyValue) types.BlockIndex {
		key := keyValue.Key
		value := keyValue.Value
		blockNumber := repo.marshaller.UnmarshallBlockKey(key)
		blockIndex := repo.marshaller.UnmarshallBlockValue(value)
		blockIndex.BlockNumber = blockNumber.String()
		return blockIndex
	}

	// there is blockNumber in REST
	if len(blockNumber) > 0 {
		key := repo.marshaller.MarshallBlockKey(blockNumber)
		keyValue, err := repo.blockDAO.FindByKey(key)
		if err != nil {
			return 0, result
		}
		result := append(result, makeBlockIndex(keyValue))
		return 1, result
	}
	// get some latest blocks
	total, keyValues := repo.blockDAO.FindByKeyPrefix([]byte(""), false, rows, start)

	for _, keyValue := range keyValues {
		result = append(result, makeBlockIndex(&keyValue))
	}
	return total, result
}
