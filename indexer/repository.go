package indexer

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/tuyennhv/geth-indexer/core/types"
)

// Repository to store index data
type Repository interface {
	Store(indexData []types.AddressIndex, blockIndex types.BlockIndex)
	Get(address string) []types.AddressIndex
	HandleReorg(blockIndex string, reorgAddresses []string)
}

// LevelDBRepo implementation of Repository
type LevelDBRepo struct {
	addressDB *leveldb.DB
	blockDB   *leveldb.DB
}

// NewLevelDBRepo create an instance of LevelDBRepo
func NewLevelDBRepo(addressDB *leveldb.DB, blockDB *leveldb.DB) *LevelDBRepo {
	return &LevelDBRepo{addressDB: addressDB, blockDB: blockDB}
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
		reorgAddresses := UnmarshallBlockDBValue(reorgAddressesByteArr)
		if reorgAddresses != nil {
			repo.HandleReorg(blockIndex.BlockNumber, reorgAddresses)
		}
	}
	for _, item := range indexData {
		// fmt.Println(item)
		batch.Put(MarshallKey(item), MarshallValue(item))
	}
	err := repo.addressDB.Write(batch, nil)
	if err != nil {
		// TODO
		fmt.Println("Cannot write to address leveldb")
	}
	err = repo.blockDB.Put([]byte(blockIndex.BlockNumber), MarshallBlockDBValue(blockIndex), nil)
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
		addressIndex := UnmarshallValue(value)
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
func (repo *LevelDBRepo) HandleReorg(blockIndex string, reorgAddresses []string) {
	batch := new(leveldb.Batch)
	for _, address := range reorgAddresses {
		addressIndexKey := marshallKey(address, blockIndex)
		batch.Delete([]byte(addressIndexKey))
	}
	err := repo.addressDB.Write(batch, nil)
	if err != nil {
		// TODO
		fmt.Println("Cannot remove old address index")
	}
}

// MarshallBlockDBValue marshall a blockIndex to []byte so that we store it as value in Block db
func MarshallBlockDBValue(blockIndex types.BlockIndex) []byte {
	value := strings.Join(blockIndex.Addresses, "_")
	return []byte(value)
}

// UnmarshallBlockDBValue unmarshall a byte array into array of address, this is for Block db
func UnmarshallBlockDBValue(value []byte) []string {
	valueStr := string(value)
	valueArr := strings.Split(valueStr, "_")
	return valueArr
}

// MarshallKey create LevelDB key
func MarshallKey(index types.AddressIndex) []byte {
	return marshallKey(index.Address, index.BlockNumber.String())
}

func marshallKey(address string, blockNumber string) []byte {
	key := address + "_" + blockNumberWidPad(blockNumber)
	key = strings.ToUpper(key)
	return []byte(key)
}

// MarshallValue create LevelDB value
func MarshallValue(index types.AddressIndex) []byte {
	value := index.TxHash + "_" + index.Value.String() + "_" + index.Time.String()
	return []byte(value)
}

// UnmarshallKey LevelDB key to address_blockNumber
func UnmarshallKey(key []byte) (string, *big.Int) {
	keyStr := string(key)
	keyArr := strings.Split(keyStr, "_")
	blockNumber := stringToBigInt(keyArr[1])
	return keyArr[0], blockNumber
}

// UnmarshallValue LevelDB value to txhash_Value_Time
func UnmarshallValue(value []byte) types.AddressIndex {
	valueStr := string(value)
	valueArr := strings.Split(valueStr, "_")
	txHash := valueArr[0]
	txValue := stringToBigInt(valueArr[1])
	time := stringToBigInt(valueArr[2])
	return types.AddressIndex{
		TxHash: txHash,
		Value:  *txValue,
		Time:   *time,
	}
}

func stringToBigInt(str string) *big.Int {
	result := new(big.Int)
	result, _ = result.SetString(str, 10)
	return result
}

func blockNumberWidPad(blockNumber string) string {
	if len(blockNumber) < 10 {
		count := 10 - len(blockNumber)
		for i := 0; i < count; i++ {
			blockNumber = "0" + blockNumber
		}
	}
	return blockNumber
}
