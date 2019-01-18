package keyvalue

import (
	"errors"
	"math/big"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue/dao"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue/marshal"
)

// BatchRepo repository for batch status
type BatchRepo interface {
	GetAllBatchStatuses() []types.BatchStatus
	UpdateBatch(batch types.BatchStatus) error
	ReplaceBatch(from *big.Int, newTo *big.Int) error
}

// KVBatchRepo implement BatchRepo
type KVBatchRepo struct {
	batchDAO   dao.KeyValueDAO
	marshaller marshal.Marshaller
}

// NewKVBatchRepo new KVBatchRepo instance
func NewKVBatchRepo(batchDAO dao.KeyValueDAO) *KVBatchRepo {
	return &KVBatchRepo{
		batchDAO:   batchDAO,
		marshaller: marshal.ByteMarshaller{},
	}
}

// GetAllBatchStatuses get all batches
func (repo *KVBatchRepo) GetAllBatchStatuses() []types.BatchStatus {
	keyValues := repo.batchDAO.GetAllRecords()
	batches := []types.BatchStatus{}
	for _, keyValue := range keyValues {
		batch := repo.keyValueToBatchStatus(keyValue)
		batches = append(batches, batch)
	}
	return batches
}

func (repo *KVBatchRepo) keyValueToBatchStatus(keyValue dao.KeyValue) types.BatchStatus {
	key := keyValue.Key
	value := keyValue.Value
	batch1 := repo.marshaller.UnmarshallBatchKey(key)
	batch2 := repo.marshaller.UnmarshallBatchValue(value)
	batch := types.BatchStatus{
		From:      batch1.From,
		To:        batch1.To,
		Step:      batch1.Step,
		CreatedAt: batch1.CreatedAt,
		UpdatedAt: batch2.UpdatedAt,
		Current:   batch2.Current,
	}
	return batch
}

// UpdateBatch update a batch
func (repo *KVBatchRepo) UpdateBatch(batch types.BatchStatus) error {
	if batch.From == nil || batch.To == nil || batch.Step <= 0 || batch.CreatedAt == nil {
		return errors.New("Batch is not valid, value:" + batch.String())
	}
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To, batch.Step, batch.CreatedAt)
	value := repo.marshaller.MarshallBatchValue(batch.UpdatedAt, batch.Current)
	return repo.batchDAO.Put(dao.NewKeyValue(key, value))
}

// ReplaceBatch replace a batch with new "to"
func (repo *KVBatchRepo) ReplaceBatch(from *big.Int, newTo *big.Int) error {
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

func (repo *KVBatchRepo) replaceBatch(batch types.BatchStatus, newTo *big.Int) error {
	key := repo.marshaller.MarshallBatchKey(batch.From, batch.To, batch.Step, batch.CreatedAt)
	repo.batchDAO.DeleteByKey(key)
	batch.To = newTo
	return repo.UpdateBatch(batch)
}

func (repo *KVBatchRepo) getBatchStatus(key []byte, value []byte) types.BatchStatus {
	batch := repo.marshaller.UnmarshallBatchKey(key)
	batchValue := repo.marshaller.UnmarshallBatchValue(value)
	batch.Current = batchValue.Current
	batch.UpdatedAt = batchValue.UpdatedAt
	return batch
}
