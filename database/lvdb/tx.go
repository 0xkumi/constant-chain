package lvdb

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/constant-money/constant-chain/common"

	"github.com/constant-money/constant-chain/database"

	"math/big"

	"github.com/pkg/errors"
	lvdberr "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// StoreSerialNumbers - store list serialNumbers by shardID
func (db *db) StoreSerialNumbers(tokenID *common.Hash, serialNumber []byte, shardID byte) error {
	key := db.GetKey(string(serialNumbersPrefix), tokenID)
	key = append(key, shardID)
	res, err := db.lvdb.Get(key, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}

	var arrayData [][]byte
	if len(res) > 0 {
		if err := json.Unmarshal(res, &arrayData); err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Unmarshal"))
		}
	}

	lenData := int64(len(arrayData))
	newIndex := big.NewInt(lenData).Bytes()
	if lenData == 0 {
		newIndex = []byte{0}
	}
	//keySpec1 := make([]byte, len(key))
	keySpec1 := append(key, serialNumber...)
	if err := db.lvdb.Put(keySpec1, newIndex, nil); err != nil {
		return err
	}

	arrayData = append(arrayData, serialNumber)
	b, err := json.Marshal(arrayData)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Marshal"))
	}
	if err := db.lvdb.Put(key, b, nil); err != nil {
		return err
	}
	return nil
}

// FetchSerialNumbers - Get list SerialNumbers by shardID
func (db *db) FetchSerialNumbers(tokenID *common.Hash, shardID byte) ([][]byte, error) {
	key := db.GetKey(string(serialNumbersPrefix), tokenID)
	key = append(key, shardID)
	res, err := db.lvdb.Get(key, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return make([][]byte, 0), database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}

	var arrayData [][]byte
	if len(res) > 0 {
		if err := json.Unmarshal(res, &arrayData); err != nil {
			return make([][]byte, 0), errors.Wrap(err, "json.Unmarshal")
		}
	}
	return arrayData, nil
}

// HasSerialNumber - Check serialNumber in list SerialNumbers by shardID
func (db *db) HasSerialNumber(tokenID *common.Hash, serialNumber []byte, shardID byte) (bool, error) {
	key := db.GetKey(string(serialNumbersPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, serialNumber...)
	_, err := db.Get(keySpec)
	if err != nil {
		return false, nil
	} else {
		return true, nil
	}
	return false, nil
}

// HasSerialNumberIndex - Check serialNumber in list SerialNumbers by shardID
/*func (db *db) HasSerialNumberIndex(serialNumberIndex int64, shardID byte) (bool, error) {
	key := db.GetKey(string(serialNumbersPrefix), "")
	key = append(key, shardID)
	keySpec := append(key, big.NewInt(serialNumberIndex).Bytes()...)
	_, err := db.Get(keySpec)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
	return false, nil
}*/

/*func (db *db) GetSerialNumberByIndex(serialNumberIndex int64, shardID byte) ([]byte, error) {
	key := db.GetKey(string(serialNumbersPrefix), "")
	key = append(key, shardID)
	keySpec := append(key, big.NewInt(serialNumberIndex).Bytes()...)
	data, err := db.Get(keySpec)
	if err != nil {
		return data, err
	} else {
		return data, nil
	}
	return data, nil
}*/

// CleanSerialNumbers - clear all list serialNumber in DB
func (db *db) CleanSerialNumbers() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(serialNumbersPrefix), nil)
	for iter.Next() {
		err := db.lvdb.Delete(iter.Key(), nil)
		if err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "iter.Error"))
	}
	return nil
}

func (db *db) StoreOutputCoins(tokenID *common.Hash, pubkey []byte, outputcoin []byte, shardID byte) error {
	key := db.GetKey(string(outcoinsPrefix), tokenID)
	key = append(key, shardID)

	// store for pubkey:[outcoint1, outcoint2, ...]
	key = append(key, pubkey...)
	var arrDatabyPubkey [][]byte
	resByPubkey, err := db.lvdb.Get(key, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	if len(resByPubkey) > 0 {
		if err := json.Unmarshal(resByPubkey, &arrDatabyPubkey); err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Unmarshal"))
		}
	}
	arrDatabyPubkey = append(arrDatabyPubkey, outputcoin)
	resByPubkey, err = json.Marshal(arrDatabyPubkey)
	if err != nil {
		return err
	}
	if err := db.lvdb.Put(key, resByPubkey, nil); err != nil {
		return err
	}

	return nil
}

// StoreCommitments - store list commitments by shardID
func (db *db) StoreCommitments(tokenID *common.Hash, pubkey []byte, commitments []byte, shardID byte) error {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	res, err := db.lvdb.Get(key, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}

	var arrData [][]byte
	if len(res) > 0 {
		if err := json.Unmarshal(res, &arrData); err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Unmarshal"))
		}
	}

	// use for create proof random
	lenData := uint64(len(arrData))
	newIndex := new(big.Int).SetUint64(lenData).Bytes()
	if lenData == 0 {
		newIndex = []byte{0}
	}
	//keySpec1 := make([]byte, len(key))
	keySpec1 := append(key, newIndex...)
	if err := db.lvdb.Put(keySpec1, commitments, nil); err != nil {
		return err
	}

	// use for validate
	//keySpec2 := make([]byte, len(key))
	keySpec2 := append(key, commitments...)
	if err := db.lvdb.Put(keySpec2, newIndex, nil); err != nil {
		return err
	}

	// store length of array commitment
	//keySpec3 := make([]byte, len(key))
	keySpec3 := append(key, []byte("len")...)
	if err := db.lvdb.Put(keySpec3, newIndex, nil); err != nil {
		return err
	}

	// store for pubkey:[newindex1, newindex2]
	//keySpec4 := make([]byte, len(key))
	keySpec4 := append(key, pubkey...)
	var arrDatabyPubkey [][]byte
	resByPubkey, err := db.lvdb.Get(keySpec4, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	if len(resByPubkey) > 0 {
		if err := json.Unmarshal(resByPubkey, &arrDatabyPubkey); err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Unmarshal"))
		}
	}
	arrDatabyPubkey = append(arrDatabyPubkey, newIndex)
	resByPubkey, err = json.Marshal(arrDatabyPubkey)
	if err != nil {
		return err
	}
	if err := db.lvdb.Put(keySpec4, resByPubkey, nil); err != nil {
		return err
	}

	arrData = append(arrData, commitments)
	b, err := json.Marshal(arrData)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Marshal"))
	}
	if err := db.lvdb.Put(key, b, nil); err != nil {
		return err
	}
	return nil
}

// FetchCommitments - Get list commitments by shardID
func (db *db) FetchCommitments(tokenID *common.Hash, shardID byte) ([][]byte, error) {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	res, err := db.lvdb.Get(key, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return make([][]byte, 0), database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}

	var txs [][]byte
	if len(res) > 0 {
		if err := json.Unmarshal(res, &txs); err != nil {
			return make([][]byte, 0), database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Unmarshal"))
		}
	}
	return txs, nil
}

// HasCommitment - Check commitment in list commitments by shardID
func (db *db) HasCommitment(tokenID *common.Hash, commitment []byte, shardID byte) (bool, error) {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, commitment...)
	_, err := db.Get(keySpec)
	if err != nil {
		return false, nil
	} else {
		return true, nil
	}
	return false, nil
}

func (db *db) HasCommitmentIndex(tokenID *common.Hash, commitmentIndex uint64, shardID byte) (bool, error) {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, new(big.Int).SetUint64(commitmentIndex).Bytes()...)
	_, err := db.Get(keySpec)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
	return false, nil
}

func (db *db) GetCommitmentByIndex(tokenID *common.Hash, commitmentIndex uint64, shardID byte) ([]byte, error) {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	//keySpec := make([]byte, len(key))
	var keySpec []byte
	if commitmentIndex == 0 {
		keySpec = append(key, byte(0))
	} else {
		keySpec = append(key, new(big.Int).SetUint64(commitmentIndex).Bytes()...)
	}
	data, err := db.Get(keySpec)
	if err != nil {
		return data, err
	} else {
		return data, nil
	}
	return data, nil
}

// GetCommitmentIndex - return index of commitment in db list
func (db *db) GetCommitmentIndex(tokenID *common.Hash, commitment []byte, shardID byte) (*big.Int, error) {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, commitment...)
	data, err := db.Get(keySpec)
	if err != nil {
		return nil, err
	} else {
		return new(big.Int).SetBytes(data), nil
	}
	return nil, nil
}

// GetCommitmentIndex - return index of commitment in db list
func (db *db) GetCommitmentLength(tokenID *common.Hash, shardID byte) (*big.Int, error) {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, []byte("len")...)
	data, err := db.Get(keySpec)
	if err != nil {
		return nil, err
	} else {
		lenArray := new(big.Int).SetBytes(data)
		lenArray = lenArray.Add(lenArray, new(big.Int).SetInt64(1))
		return lenArray, nil
	}
	return nil, nil
}

func (db *db) GetCommitmentIndexsByPubkey(tokenID *common.Hash, pubkey []byte, shardID byte) ([][]byte, error) {
	key := db.GetKey(string(commitmentsPrefix), tokenID)
	key = append(key, shardID)

	//keySpec4 := make([]byte, len(key))
	keySpec4 := append(key, pubkey...)
	var arrDatabyPubkey [][]byte
	resByPubkey, err := db.lvdb.Get(keySpec4, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	if len(resByPubkey) > 0 {
		if err := json.Unmarshal(resByPubkey, &arrDatabyPubkey); err != nil {
			return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Unmarshal"))
		}
	}
	return arrDatabyPubkey, nil
}

func (db *db) GetOutcoinsByPubkey(tokenID *common.Hash, pubkey []byte, shardID byte) ([][]byte, error) {
	key := db.GetKey(string(outcoinsPrefix), tokenID)
	key = append(key, shardID)

	key = append(key, pubkey...)
	var arrDatabyPubkey [][]byte
	resByPubkey, err := db.lvdb.Get(key, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	if len(resByPubkey) > 0 {
		if err := json.Unmarshal(resByPubkey, &arrDatabyPubkey); err != nil {
			return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "json.Unmarshal"))
		}
	}
	return arrDatabyPubkey, nil
}

// CleanCommitments - clear all list commitments in DB
func (db *db) CleanCommitments() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(commitmentsPrefix), nil)
	for iter.Next() {
		err := db.lvdb.Delete(iter.Key(), nil)
		if err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "iter.Error"))
	}
	return nil
}

// StoreSNDerivators - store list serialNumbers by shardID
func (db *db) StoreSNDerivators(tokenID *common.Hash, data []byte, shardID byte) error {
	key := db.GetKey(string(snderivatorsPrefix), tokenID)
	key = append(key, shardID)
	_, err := db.lvdb.Get(key, nil)
	if err != nil && err != lvdberr.ErrNotFound {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}

	// "snderivator-data:nil"
	keySpec := append(key, data...)
	if err := db.lvdb.Put(keySpec, []byte{}, nil); err != nil {
		return err
	}

	return nil
}

// HasSNDerivator - Check SnDerivator in list SnDerivators by shardID
func (db *db) HasSNDerivator(tokenID *common.Hash, data []byte, shardID byte) (bool, error) {
	key := db.GetKey(string(snderivatorsPrefix), tokenID)
	key = append(key, shardID)
	keySpec := append(key, data...)
	_, err := db.Get(keySpec)
	if err != nil {
		return false, nil
	} else {
		return true, nil
	}
	return false, nil
}

// CleanCommitments - clear all list commitments in DB
func (db *db) CleanSNDerivator() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(snderivatorsPrefix), nil)
	for iter.Next() {
		err := db.lvdb.Delete(iter.Key(), nil)
		if err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "iter.Error"))
	}
	return nil
}

// StoreFeeEstimator - Store data for FeeEstimator object
func (db *db) StoreFeeEstimator(val []byte, shardID byte) error {
	if err := db.Put(append(feeEstimator, shardID), val); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.Put"))
	}
	return nil
}

// GetFeeEstimator - Get data for FeeEstimator object as a json in byte format
func (db *db) GetFeeEstimator(shardID byte) ([]byte, error) {
	b, err := db.lvdb.Get(append(feeEstimator, shardID), nil)
	if err != nil {
		return nil, database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Get"))
	}
	return b, err
}

// CleanFeeEstimator - Clear FeeEstimator
func (db *db) CleanFeeEstimator() error {
	iter := db.lvdb.NewIterator(util.BytesPrefix(feeEstimator), nil)
	for iter.Next() {
		err := db.lvdb.Delete(iter.Key(), nil)
		if err != nil {
			return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Delete"))
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "iter.Error"))
	}
	return nil
}

/*
	StoreTransactionIndex
	Store tx detail location
  Key: prefixTx-txHash
	H: blockHash-blockIndex
*/
func (db *db) StoreTransactionIndex(txId *common.Hash, blockHash *common.Hash, index int) error {
	key := string(transactionKeyPrefix) + txId.String()
	value := blockHash.String() + string(Splitter) + strconv.Itoa(index)
	if err := db.lvdb.Put([]byte(key), []byte(value), nil); err != nil {
		return err
	}

	return nil
}

/*
  Get Transaction by ID
*/

func (db *db) GetTransactionIndexById(txId *common.Hash) (*common.Hash, int, *database.DatabaseError) {
	key := string(transactionKeyPrefix) + txId.String()
	_, err := db.HasValue([]byte(key))
	if err != nil {
		return nil, -1, database.NewDatabaseError(database.ErrUnexpected, err)
	}

	res, err := db.Get([]byte(key))
	if err != nil {
		return nil, -1, database.NewDatabaseError(database.ErrUnexpected, err)
	}
	reses := strings.Split(string(res), (string(Splitter)))
	hash, err := common.Hash{}.NewHashFromStr(reses[0])
	if err != nil {
		return nil, -1, database.NewDatabaseError(database.ErrUnexpected, err)
	}
	index, err := strconv.Atoi(reses[1])
	if err != nil {
		return nil, -1, database.NewDatabaseError(database.ErrUnexpected, err)
	}
	return hash, index, nil
}
