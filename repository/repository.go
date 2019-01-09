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
	Store(indexData []*types.AddressIndex, blockIndex *types.BlockIndex, isBatch bool) error
	GetTransactionByAddress(address string, rows int, start int) (int, []types.AddressIndex)
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
			time, reorgAddresses := repo.marshaller.UnmarshallBlockValue(oldBlock.Value)
			if reorgAddresses != nil && len(reorgAddresses) > 0 {
				err = repo.HandleReorg(time, reorgAddresses)
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
func (repo *LevelDBRepo) GetTransactionByAddress(address string, rows int, start int) (int, []types.AddressIndex) {
	result := []types.AddressIndex{}
	prefix := repo.marshaller.MarshallAddressKeyPrefix(address)
	if len(prefix) == 0 {
		return 0, result
	}
	asc := false
	total, keyValues := repo.addressDAO.FindByKeyPrefix(prefix, asc, rows, start)
	for _, keyValue := range keyValues {
		addressIndex := repo.keyValueToAddressIndex(keyValue)
		result = append(result, addressIndex)
	}

	return total, result
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
func (repo *LevelDBRepo) HandleReorg(time *big.Int, reorgAddresses []types.AddressSequence) error {
	keys := [][]byte{}
	for _, address := range reorgAddresses {
		// Block database save address and max sequence as value
		for i := uint8(1); i <= address.Sequence; i++ {
			addressIndexKey := repo.marshaller.MarshallAddressKeyStr(address.Address, time, i)
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
func (repo *LevelDBRepo) UpdateBatch(batch types.BatchStatus) error {
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To)
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
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To)
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
		time, reorgAddresses := repo.marshaller.UnmarshallBlockValue(value)
		return types.BlockIndex{
			BlockNumber: blockNumber.String(),
			Addresses:   reorgAddresses,
			Time:        time,
		}
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
