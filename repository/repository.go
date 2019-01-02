package repository

import (
	"fmt"

	"github.com/WeTrustPlatform/account-indexer/core/types"
	"github.com/WeTrustPlatform/account-indexer/repository/marshal"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Repository to store index data
type Repository interface {
	Store(indexData []types.AddressIndex, blockIndex types.BlockIndex)
	Get(address string) []types.AddressIndex
	HandleReorg(blockIndex string, reorgAddresses []types.AddressSequence)
}

// LevelDBRepo implementation of Repository
type LevelDBRepo struct {
	addressDB  *leveldb.DB
	blockDB    *leveldb.DB
	marshaller marshal.Marshaller
}

// NewLevelDBRepo create an instance of LevelDBRepo
func NewLevelDBRepo(addressDB *leveldb.DB, blockDB *leveldb.DB) *LevelDBRepo {
	// return &LevelDBRepo{addressDB: addressDB, blockDB: blockDB, marshaller: marshal.StringMarshaller{}}
	return &LevelDBRepo{addressDB: addressDB, blockDB: blockDB, marshaller: marshal.ByteMarshaller{}}
}

// Store implements Repository
func (repo *LevelDBRepo) Store(indexData []types.AddressIndex, blockIndex types.BlockIndex) {
	batch := new(leveldb.Batch)
	reorgAddressesByteArr, _ := repo.blockDB.Get([]byte(blockIndex.BlockNumber), nil)
	// if err != nil {
	// 	// not found is also an error so ignorning it
	// 	fmt.Println("Cannot access block leveldb, error=" + err.Error())
	// }
	if reorgAddressesByteArr != nil {
		reorgAddresses := repo.marshaller.UnmarshallBlockDBValue(reorgAddressesByteArr)
		if reorgAddresses != nil {
			repo.HandleReorg(blockIndex.BlockNumber, reorgAddresses)
		}
	}
	for _, item := range indexData {
		// fmt.Println(item)
		batch.Put(repo.marshaller.MarshallAddressKey(item), repo.marshaller.MarshallAddressValue(item))
	}
	err := repo.addressDB.Write(batch, nil)
	if err != nil {
		// TODO
		fmt.Println("Cannot write to address leveldb")
	}
	err = repo.blockDB.Put([]byte(blockIndex.BlockNumber), repo.marshaller.MarshallBlockDBValue(blockIndex), nil)
	if err != nil {
		// TODO
		fmt.Println("Cannot write to block leveldb")
	}
}

// Get get transaction list from an address
func (repo *LevelDBRepo) Get(address string) []types.AddressIndex {
	result := []types.AddressIndex{}
	iter := repo.addressDB.NewIterator(util.BytesPrefix([]byte(address)), nil)
	for iter.Next() {
		value := iter.Value()
		addressIndex := repo.marshaller.UnmarshallAddressValue(value)
		addressIndex.Address = address
		// missing block index, do we need it?
		result = append(result, addressIndex)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		fmt.Println("Cannot get address info from address DB")
	}
	return result
}

// HandleReorg handle reorg scenario: get block again
func (repo *LevelDBRepo) HandleReorg(blockIndex string, reorgAddresses []types.AddressSequence) {
	batch := new(leveldb.Batch)
	for _, address := range reorgAddresses {
		addressIndexKey := repo.marshaller.MarshallAddressKeyStr(address.Address, blockIndex, address.Sequence)
		batch.Delete([]byte(addressIndexKey))
	}
	err := repo.addressDB.Write(batch, nil)
	if err != nil {
		// TODO
		fmt.Println("Cannot remove old address index")
	}
}
