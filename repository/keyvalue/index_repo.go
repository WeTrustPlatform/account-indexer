package keyvalue

import (
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue/dao"
	"github.com/WeTrustPlatform/account-indexer/repository/keyvalue/marshal"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// KVIndexRepo implementation of IndexRepo
type KVIndexRepo struct {
	addressDAO dao.KeyValueDAO
	blockDAO   dao.KeyValueDAO
	marshaller marshal.Marshaller
}

// NewKVIndexRepo create an instance of KVIndexRepo
func NewKVIndexRepo(addressDAO dao.KeyValueDAO, blockDAO dao.KeyValueDAO) *KVIndexRepo {
	return &KVIndexRepo{
		addressDAO: addressDAO,
		blockDAO:   blockDAO,
		marshaller: marshal.ByteMarshaller{},
	}
}

// Store implements IndexRepo
func (repo *KVIndexRepo) Store(addressIndex []*types.AddressIndex, blockIndex *types.BlockIndex, isBatch bool) error {
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
func (repo *KVIndexRepo) SaveAddressIndex(addressIndex []*types.AddressIndex) error {
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
func (repo *KVIndexRepo) SaveBlockIndex(blockIndex *types.BlockIndex) error {
	key := repo.marshaller.MarshallBlockKey(blockIndex.BlockNumber)
	value := repo.marshaller.MarshallBlockValue(blockIndex)
	err := repo.blockDAO.Put(dao.NewKeyValue(key, value))
	if err != nil {
		log.Fatal("Cannot write to block leveldb")
	}
	return err
}

// GetTransactionByAddress main thing for this indexer
func (repo *KVIndexRepo) GetTransactionByAddress(address string, rows int, start int, fromTime time.Time, toTime time.Time) (int, []types.AddressIndex) {
	convertKeyValuesToAddressIndexes := func(keyValues []dao.KeyValue) []types.AddressIndex {
		result := []types.AddressIndex{}
		for _, keyValue := range keyValues {
			addressIndex := repo.keyValueToAddressIndex(keyValue)
			result = append(result, addressIndex)
		}
		return result
	}

	prefix := repo.marshaller.MarshallAddressKeyPrefix(address)
	hasFrom := !time.Time.IsZero(fromTime)
	hasTo := !time.Time.IsZero(toTime)
	if hasFrom || hasTo {
		// assuming fromTime and toTime is good
		// make toTime inclusive
		allTimeRange := util.BytesPrefix(prefix)
		fromPrefix := repo.marshaller.MarshallAddressKeyPrefix3(address, fromTime)
		newToTime := toTime.Add(1 * time.Second)
		toPrefix := repo.marshaller.MarshallAddressKeyPrefix3(address, newToTime)
		var rg *util.Range
		if !hasFrom {
			rg = &util.Range{Start: allTimeRange.Start, Limit: toPrefix}
		} else if !hasTo {
			rg = &util.Range{Start: fromPrefix, Limit: allTimeRange.Limit}
		} else {
			rg = &util.Range{Start: fromPrefix, Limit: toPrefix}
		}

		asc := true
		total, keyValues := repo.addressDAO.FindByRange(rg, asc, rows, start)
		addressIndexes := convertKeyValuesToAddressIndexes(keyValues)
		return total, addressIndexes
	}
	// Search by address as LevelDB prefix

	// bad address
	if len(prefix) == 0 {
		return 0, []types.AddressIndex{}
	}
	asc := false
	total, keyValues := repo.addressDAO.FindByKeyPrefix(prefix, asc, rows, start)
	addressIndexes := convertKeyValuesToAddressIndexes(keyValues)

	return total, addressIndexes
}

func (repo *KVIndexRepo) keyValueToAddressIndex(keyValue dao.KeyValue) types.AddressIndex {
	value := keyValue.Value
	addressIndex := repo.marshaller.UnmarshallAddressValue(value)
	key := keyValue.Key
	address, time := repo.marshaller.UnmarshallAddressKey(key)
	addressIndex.Address = address
	addressIndex.Time = time
	return addressIndex
}

// HandleReorg handle reorg scenario: get block again
func (repo *KVIndexRepo) HandleReorg(blockTime *big.Int, reorgAddresses []types.AddressSequence) error {
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

// GetLastBlock latest saved block in newHead block DB
func (repo *KVIndexRepo) GetLastBlock() (types.BlockIndex, error) {
	lastBlocks := repo.blockDAO.GetNLastRecords(1)
	if len(lastBlocks) <= 0 {
		return types.BlockIndex{}, errors.New("no last record")
	}
	return repo.keyValueToBlockIndex(lastBlocks[0]), nil
}

// GetFirstBlock first saved block in newHead block DB
func (repo *KVIndexRepo) GetFirstBlock() (types.BlockIndex, error) {
	firstBlock := repo.blockDAO.GetNFirstRecords(1)
	if len(firstBlock) <= 0 {
		return types.BlockIndex{}, errors.New("no first record")
	}
	return repo.keyValueToBlockIndex(firstBlock[0]), nil
}

// DeleteOldBlocks delete blocks where CreatedAt < untilTime
func (repo *KVIndexRepo) DeleteOldBlocks(untilTime *big.Int) (int, error) {
	pre := func(keyValue dao.KeyValue) bool {
		blockIndex := repo.keyValueToBlockIndex(keyValue)
		return blockIndex.CreatedAt.Cmp(untilTime) < 0
	}
	kvsToDel := repo.blockDAO.GetNFirstPredicate(pre)
	keys := [][]byte{}
	for _, kv := range kvsToDel {
		keys = append(keys, kv.Key)
	}
	err := repo.blockDAO.BatchDelete(keys)
	return len(kvsToDel), err
}

func (repo *KVIndexRepo) keyValueToBlockIndex(keyValue dao.KeyValue) types.BlockIndex {
	key := keyValue.Key
	blockNumber := repo.marshaller.UnmarshallBlockKey(key)
	value := keyValue.Value
	blockIndex := repo.marshaller.UnmarshallBlockValue(value)
	blockIndex.BlockNumber = blockNumber.String()
	return blockIndex
}

// GetBlocks by blockNumber. blockNumber = blank => latest block
func (repo *KVIndexRepo) GetBlocks(blockNumber string, rows int, start int) (int, []types.BlockIndex) {
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
