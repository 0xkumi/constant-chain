package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/constant-money/constant-chain/cashec"
	"github.com/constant-money/constant-chain/common"
	"github.com/constant-money/constant-chain/common/base58"
	"github.com/constant-money/constant-chain/database"
	"github.com/constant-money/constant-chain/database/lvdb"
	"github.com/constant-money/constant-chain/metadata"
	"github.com/constant-money/constant-chain/privacy"
	"github.com/constant-money/constant-chain/transaction"
	libp2p "github.com/libp2p/go-libp2p-peer"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

/*
blockChain is a view presents for data in blockchain network
because we use 20 chain data to contain all block in system, so
this struct has a array best state with len = 20,
every beststate present for a best block in every chain
*/
type BlockChain struct {
	BestState *BestState
	config    Config
	chainLock sync.Mutex
	//channel
	cQuitSync  chan struct{}
	syncStatus struct {
		Beacon bool
		Shards map[byte]struct{}
		sync.Mutex

		CurrentlySyncShardBlkByHash           map[byte]*cache.Cache
		CurrentlySyncShardBlkByHeight         map[byte]*cache.Cache
		CurrentlySyncBeaconBlkByHash          *cache.Cache
		CurrentlySyncBeaconBlkByHeight        *cache.Cache
		CurrentlySyncShardToBeaconBlkByHash   map[byte]*cache.Cache
		CurrentlySyncShardToBeaconBlkByHeight map[byte]*cache.Cache
		CurrentlySyncCrossShardBlkByHash      map[byte]*cache.Cache
		CurrentlySyncCrossShardBlkByHeight    map[byte]*cache.Cache

		PeersState     map[libp2p.ID]*peerState
		PeersStateLock sync.Mutex
		IsReady        struct {
			sync.Mutex
			Beacon bool
			Shards map[byte]bool
		}
	}
	ConsensusOngoing bool
}
type BestState struct {
	Beacon *BestStateBeacon
	Shard  map[byte]*BestStateShard
}

// config is a descriptor which specifies the blockchain instance configuration.
type Config struct {
	DataBase                  database.DatabaseInterface
	Interrupt                 <-chan struct{}
	ChainParams               *Params
	RelayShards               []byte
	NodeMode                  string
	customTokenRewardSnapshot map[string]uint64 //snapshot reward
	ShardToBeaconPool         ShardToBeaconPool
	CrossShardPool            map[byte]CrossShardPool
	BeaconPool                BeaconPool
	ShardPool                 map[byte]ShardPool
	TxPool                    TxPool
	TempTxPool                TxPool
	CRemovedTxs               chan metadata.Transaction
	Server                    interface {
		BoardcastNodeState() error

		PushMessageGetBlockBeaconByHeight(from uint64, to uint64, peerID libp2p.ID) error
		PushMessageGetBlockBeaconByHash(blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error

		PushMessageGetBlockShardByHeight(shardID byte, from uint64, to uint64, peerID libp2p.ID) error
		PushMessageGetBlockShardByHash(shardID byte, blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error

		PushMessageGetBlockShardToBeaconByHeight(shardID byte, from uint64, to uint64, peerID libp2p.ID) error
		PushMessageGetBlockShardToBeaconByHash(shardID byte, blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error
		PushMessageGetBlockShardToBeaconBySpecificHeight(shardID byte, blksHeight []uint64, getFromPool bool, peerID libp2p.ID) error

		PushMessageGetBlockCrossShardByHash(fromShard byte, toShard byte, blksHash []common.Hash, getFromPool bool, peerID libp2p.ID) error
		PushMessageGetBlockCrossShardBySpecificHeight(fromShard byte, toShard byte, blksHeight []uint64, getFromPool bool, peerID libp2p.ID) error
		UpdateConsensusState(role string, userPbk string, currentShard *byte, beaconCommittee []string, shardCommittee map[byte][]string)
	}
	UserKeySet *cashec.KeySet
}

/*
Init - init a blockchain view from config
*/
func (blockchain *BlockChain) Init(config *Config) error {
	// Enforce required config fields.
	if config.DataBase == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Database is not config"))
	}
	if config.ChainParams == nil {
		return NewBlockChainError(UnExpectedError, errors.New("Chain parameters is not config"))
	}

	blockchain.config = *config

	// Initialize the chain state from the passed database.  When the db
	// does not yet contain any chain state, both it and the chain state
	// will be initialized to contain only the genesis block.
	if err := blockchain.initChainState(); err != nil {
		return err
	}

	blockchain.cQuitSync = make(chan struct{})
	blockchain.syncStatus.Shards = make(map[byte]struct{})
	blockchain.syncStatus.PeersState = make(map[libp2p.ID]*peerState)
	blockchain.syncStatus.IsReady.Shards = make(map[byte]bool)
	return nil
}

func (blockchain *BlockChain) AddTxPool(txpool TxPool) {
	blockchain.config.TxPool = txpool
}

func (blockchain *BlockChain) AddTempTxPool(temptxpool TxPool) {
	blockchain.config.TempTxPool = temptxpool
}

func (blockchain *BlockChain) InitChannelBlockchain(cRemovedTxs chan metadata.Transaction) {
	blockchain.config.CRemovedTxs = cRemovedTxs
}

// -------------- Blockchain retriever's implementation --------------
// GetCustomTokenTxsHash - return list of tx which relate to custom token
func (blockchain *BlockChain) GetCustomTokenTxs(tokenID *common.Hash) (map[common.Hash]metadata.Transaction, error) {
	txHashesInByte, err := blockchain.config.DataBase.CustomTokenTxs(tokenID)
	if err != nil {
		return nil, err
	}
	result := make(map[common.Hash]metadata.Transaction)
	for _, temp := range txHashesInByte {
		_, _, _, tx, err := blockchain.GetTransactionByHash(temp)
		if err != nil {
			return nil, err
		}
		result[*tx.Hash()] = tx
	}
	return result, nil
}

// -------------- End of Blockchain retriever's implementation --------------

/*
// initChainState attempts to load and initialize the chain state from the
// database.  When the db does not yet contain any chain state, both it and the
// chain state are initialized to the genesis block.
*/
func (blockchain *BlockChain) initChainState() error {
	// Determine the state of the chain database. We may need to initialize
	// everything from scratch or upgrade certain buckets.
	var initialized bool

	blockchain.BestState = &BestState{
		Beacon: nil,
		Shard:  make(map[byte]*BestStateShard),
	}

	bestStateBeaconBytes, err := blockchain.config.DataBase.FetchBeaconBestState()
	if err == nil {
		beacon := &BestStateBeacon{}
		err = json.Unmarshal(bestStateBeaconBytes, beacon)
		//update singleton object
		SetBestStateBeacon(beacon)
		//update beacon field in blockchain Beststate
		blockchain.BestState.Beacon = GetBestStateBeacon()

		if err != nil {
			initialized = false
		} else {
			initialized = true
		}
	} else {
		initialized = false
	}
	if !initialized {
		// At this point the database has not already been initialized, so
		// initialize both it and the chain state to the genesis block.
		err := blockchain.initBeaconState()
		if err != nil {
			return err
		}

	}

	for shard := 1; shard <= blockchain.BestState.Beacon.ActiveShards; shard++ {
		shardID := byte(shard - 1)
		bestStateBytes, err := blockchain.config.DataBase.FetchShardBestState(shardID)
		if err == nil {
			shardBestState := &BestStateShard{}
			err = json.Unmarshal(bestStateBytes, shardBestState)
			//update singleton object
			SetBestStateShard(shardID, shardBestState)
			//update Shard field in blockchain Beststate
			blockchain.BestState.Shard[shardID] = GetBestStateShard(shardID)
			if err != nil {
				initialized = false
			} else {
				initialized = true
			}
		} else {
			initialized = false
		}

		if !initialized {
			// At this point the database has not already been initialized, so
			// initialize both it and the chain state to the genesis block.
			err := blockchain.initShardState(shardID)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

/*
// createChainState initializes both the database and the chain state to the
// genesis block.  This includes creating the necessary buckets and inserting
// the genesis block, so it must only be called on an uninitialized database.
*/
func (blockchain *BlockChain) initShardState(shardID byte) error {
	blockchain.BestState.Shard[shardID] = InitBestStateShard(shardID, blockchain.config.ChainParams)
	// Create a new block from genesis block and set it as best block of chain
	initBlock := ShardBlock{}
	initBlock = *blockchain.config.ChainParams.GenesisShardBlock
	initBlock.Header.ShardID = shardID

	_, newShardCandidate := GetStakingCandidate(*blockchain.config.ChainParams.GenesisBeaconBlock)

	blockchain.BestState.Shard[shardID].ShardCommittee = append(blockchain.BestState.Shard[shardID].ShardCommittee, newShardCandidate[int(shardID)*blockchain.config.ChainParams.ShardCommitteeSize:(int(shardID)*blockchain.config.ChainParams.ShardCommitteeSize)+blockchain.config.ChainParams.ShardCommitteeSize]...)

	genesisBeaconBlk, err := blockchain.GetBeaconBlockByHeight(1)
	if err != nil {
		return NewBlockChainError(UnExpectedError, err)
	}
	err = blockchain.BestState.Shard[shardID].Update(&initBlock, []*BeaconBlock{genesisBeaconBlk})
	if err != nil {
		return err
	}
	blockchain.ProcessStoreShardBlock(&initBlock)
	return nil
}

func (blockchain *BlockChain) initBeaconState() error {
	blockchain.BestState.Beacon = InitBestStateBeacon(blockchain.config.ChainParams)
	initBlock := blockchain.config.ChainParams.GenesisBeaconBlock
	blockchain.BestState.Beacon.Update(initBlock, blockchain)

	// Insert new block into beacon chain
	if err := blockchain.StoreBeaconBestState(); err != nil {
		Logger.log.Error("Error Store best state for block", blockchain.BestState.Beacon.BestBlockHash, "in beacon chain")
		return NewBlockChainError(UnExpectedError, err)
	}
	if err := blockchain.config.DataBase.StoreBeaconBlock(&blockchain.BestState.Beacon.BestBlock); err != nil {
		Logger.log.Error("Error store beacon block", blockchain.BestState.Beacon.BestBlockHash, "in beacon chain")
		return err
	}
	if err := blockchain.config.DataBase.StoreCommitteeByEpoch(initBlock.Header.Epoch, blockchain.BestState.Beacon.GetShardCommittee()); err != nil {
		return err
	}
	blockHash := initBlock.Hash()
	if err := blockchain.config.DataBase.StoreBeaconBlockIndex(blockHash, initBlock.Header.Height); err != nil {
		return err
	}
	return nil
}

/*
Get block index(height) of block
*/
func (blockchain *BlockChain) GetBlockHeightByBlockHash(hash *common.Hash) (uint64, byte, error) {
	return blockchain.config.DataBase.GetIndexOfBlock(hash)
}

/*
Get block hash by block index(height)
*/
func (blockchain *BlockChain) GetBeaconBlockHashByHeight(height uint64) (*common.Hash, error) {
	return blockchain.config.DataBase.GetBeaconBlockHashByIndex(height)
}

/*
Fetch DatabaseInterface and get block by index(height) of block
*/
func (blockchain *BlockChain) GetBeaconBlockByHeight(height uint64) (*BeaconBlock, error) {
	hashBlock, err := blockchain.config.DataBase.GetBeaconBlockHashByIndex(height)
	if err != nil {
		return nil, err
	}
	block, _, err := blockchain.GetBeaconBlockByHash(hashBlock)
	if err != nil {
		return nil, err
	}
	return block, nil
}

/*
Fetch DatabaseInterface and get block data by block hash
*/
func (blockchain *BlockChain) GetBeaconBlockByHash(hash *common.Hash) (*BeaconBlock, uint64, error) {
	blockBytes, err := blockchain.config.DataBase.FetchBeaconBlock(hash)
	if err != nil {
		return nil, 0, err
	}
	block := BeaconBlock{}
	err = json.Unmarshal(blockBytes, &block)
	if err != nil {
		return nil, 0, err
	}
	return &block, uint64(len(blockBytes)), nil
}

/*
Get block index(height) of block
*/
func (blockchain *BlockChain) GetShardBlockHeightByHash(hash *common.Hash) (uint64, byte, error) {
	return blockchain.config.DataBase.GetIndexOfBlock(hash)
}

/*
Get block hash by block index(height)
*/
func (blockchain *BlockChain) GetShardBlockHashByHeight(height uint64, shardID byte) (*common.Hash, error) {
	return blockchain.config.DataBase.GetBlockByIndex(height, shardID)
}

/*
Fetch DatabaseInterface and get block by index(height) of block
*/
func (blockchain *BlockChain) GetShardBlockByHeight(height uint64, shardID byte) (*ShardBlock, error) {
	hashBlock, err := blockchain.config.DataBase.GetBlockByIndex(height, shardID)
	if err != nil {
		return nil, err
	}
	block, _, err := blockchain.GetShardBlockByHash(hashBlock)

	return block, err
}

/*
Fetch DatabaseInterface and get block data by block hash
*/
func (blockchain *BlockChain) GetShardBlockByHash(hash *common.Hash) (*ShardBlock, uint64, error) {
	blockBytes, err := blockchain.config.DataBase.FetchBlock(hash)
	if err != nil {
		return nil, 0, err
	}

	block := ShardBlock{}
	err = json.Unmarshal(blockBytes, &block)
	if err != nil {
		return nil, 0, err
	}
	return &block, uint64(len(blockBytes)), nil
}

/*
Store best state of block(best block, num of tx, ...) into Database
*/
func (blockchain *BlockChain) StoreBeaconBestState() error {
	return blockchain.config.DataBase.StoreBeaconBestState(blockchain.BestState.Beacon)
}

/*
Store best state of block(best block, num of tx, ...) into Database
*/
func (blockchain *BlockChain) StoreShardBestState(shardID byte) error {
	return blockchain.config.DataBase.StoreShardBestState(blockchain.BestState.Shard[shardID], shardID)
}

/*
GetBestState - return a best state from a chain
*/
// #1 - shardID - index of chain
func (blockchain *BlockChain) GetShardBestState(shardID byte) (*BestStateShard, error) {
	bestState := BestStateShard{}
	bestStateBytes, err := blockchain.config.DataBase.FetchShardBestState(shardID)
	if err == nil {
		err = json.Unmarshal(bestStateBytes, &bestState)
	}
	return &bestState, err
}

/*
Store block into Database
*/
func (blockchain *BlockChain) StoreShardBlock(block *ShardBlock) error {
	return blockchain.config.DataBase.StoreShardBlock(block, block.Header.ShardID)
}

/*
	Store Only Block Header into database
*/
func (blockchain *BlockChain) StoreShardBlockHeader(block *ShardBlock) error {
	//Logger.log.Infof("Store Block Header, block header %+v, block hash %+v, chain id %+v",block.Header, block.blockHash, block.Header.shardID)
	return blockchain.config.DataBase.StoreShardBlockHeader(block.Header, block.Hash(), block.Header.ShardID)
}

/*
Save index(height) of block by block hash
and
Save block hash by index(height) of block
*/
func (blockchain *BlockChain) StoreShardBlockIndex(block *ShardBlock) error {
	return blockchain.config.DataBase.StoreShardBlockIndex(block.Hash(), block.Header.Height, block.Header.ShardID)
}

func (blockchain *BlockChain) StoreTransactionIndex(txHash *common.Hash, blockHash *common.Hash, index int) error {
	return blockchain.config.DataBase.StoreTransactionIndex(txHash, blockHash, index)
}

/*
Uses an existing database to update the set of used tx by saving list nullifier of privacy,
this is a list tx-out which are used by a new tx
*/
func (blockchain *BlockChain) StoreSerialNumbersFromTxViewPoint(view TxViewPoint) error {
	for _, item1 := range view.listSerialNumbers {
		err := blockchain.config.DataBase.StoreSerialNumbers(view.tokenID, item1, view.shardID)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
Uses an existing database to update the set of used tx by saving list SNDerivator of privacy,
this is a list tx-out which are used by a new tx
*/
func (blockchain *BlockChain) StoreSNDerivatorsFromTxViewPoint(view TxViewPoint, shardID byte) error {
	// commitment
	keys := make([]string, 0, len(view.mapCommitments))
	for k := range view.mapCommitments {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// Store SND of every transaction in this block
		// UNCOMMENT: TO STORE SND WITH NON-CROSS SHARD TRANSACTION ONLY
		// pubkey := k
		// pubkeyBytes, _, err := base58.Base58Check{}.Decode(pubkey)
		// if err != nil {
		// 	return err
		// }
		// lastByte := pubkeyBytes[len(pubkeyBytes)-1]
		// pubkeyShardID := common.GetShardIDFromLastByte(lastByte)
		// if pubkeyShardID == shardID {
		item1 := view.mapSnD[k]
		for _, snd := range item1 {
			err := blockchain.config.DataBase.StoreSNDerivators(view.tokenID, privacy.AddPaddingBigInt(&snd, privacy.BigIntSize), view.shardID)
			if err != nil {
				return err
			}
			// }
		}
	}

	// for pubkey, items := range view.mapSnD {
	// 	pubkeyBytes, _, err := base58.Base58Check{}.Decode(pubkey)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	lastByte := pubkeyBytes[len(pubkeyBytes)-1]
	// 	pubkeyShardID := common.GetShardIDFromLastByte(lastByte)
	// 	if pubkeyShardID == shardID {
	// 		for _, item1 := range items {
	// 			err := blockchain.config.DataBase.StoreSNDerivators(view.tokenID, item1, view.shardID)
	// 			if err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}
	// }
	return nil
}

/*
Uses an existing database to update the set of not used tx by saving list commitments of privacy,
this is a list tx-in which are used by a new tx
*/
func (blockchain *BlockChain) StoreCommitmentsFromTxViewPoint(view TxViewPoint, shardID byte) error {

	// commitment
	keys := make([]string, 0, len(view.mapCommitments))
	for k := range view.mapCommitments {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		pubkey := k
		item1 := view.mapCommitments[k]
		pubkeyBytes, _, err := base58.Base58Check{}.Decode(pubkey)
		if err != nil {
			return err
		}
		lastByte := pubkeyBytes[len(pubkeyBytes)-1]
		pubkeyShardID := common.GetShardIDFromLastByte(lastByte)
		if pubkeyShardID == shardID {
			for _, com := range item1 {
				err = blockchain.config.DataBase.StoreCommitments(view.tokenID, pubkeyBytes, com, view.shardID)
				if err != nil {
					return err
				}
			}
		}
	}

	// outputs
	keys = make([]string, 0, len(view.mapOutputCoins))
	for k := range view.mapOutputCoins {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		pubkey := k
		item1 := view.mapOutputCoins[k]

		pubkeyBytes, _, err := base58.Base58Check{}.Decode(pubkey)
		if err != nil {
			return err
		}
		lastByte := pubkeyBytes[len(pubkeyBytes)-1]
		pubkeyShardID := common.GetShardIDFromLastByte(lastByte)
		if pubkeyShardID == shardID {
			for _, outcoin := range item1 {
				err = blockchain.config.DataBase.StoreOutputCoins(view.tokenID, pubkeyBytes, outcoin.Bytes(), pubkeyShardID)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// CreateAndSaveTxViewPointFromBlock - fetch data from block, put into txviewpoint variable and save into db
// @note: still storage full data of commitments, serialnumbersm snderivator to check double spend
// @note: this function only work for transaction transfer token/constant within shard
func (blockchain *BlockChain) CreateAndSaveTxViewPointFromBlock(block *ShardBlock) error {
	// Fetch data from block into tx View point
	view := NewTxViewPoint(block.Header.ShardID)
	err := view.fetchTxViewPointFromBlock(blockchain.config.DataBase, block)
	if err != nil {
		return err
	}

	// check normal custom token
	for indexTx, customTokenTx := range view.customTokenTxs {
		switch customTokenTx.TxTokenData.Type {
		case transaction.CustomTokenInit:
			{
				Logger.log.Info("Store custom token when it is issued", customTokenTx.TxTokenData.PropertyID, customTokenTx.TxTokenData.PropertySymbol, customTokenTx.TxTokenData.PropertyName)
				err = blockchain.config.DataBase.StoreCustomToken(&customTokenTx.TxTokenData.PropertyID, customTokenTx.Hash()[:])
				if err != nil {
					return err
				}
			}
		case transaction.CustomTokenCrossShard:
			{
				// 0xsirrush updated: check existed token ID
				existedToken := blockchain.CustomTokenIDExisted(&customTokenTx.TxTokenData.PropertyID)
				//If don't exist then create
				if !existedToken {
					Logger.log.Info("Store Cross Shard Custom if It's not existed in DB", customTokenTx.TxTokenData.PropertyID, customTokenTx.TxTokenData.PropertySymbol, customTokenTx.TxTokenData.PropertyName)
					err = blockchain.config.DataBase.StoreCustomToken(&customTokenTx.TxTokenData.PropertyID, customTokenTx.Hash()[:])
					if err != nil {
						Logger.log.Error("CreateAndSaveTxViewPointFromBlock", err)
					}
				}
				/*listCustomToken, err := blockchain.ListCustomToken()
				if err != nil {
					panic(err)
				}
				//If don't exist then create
				if _, ok := listCustomToken[customTokenTx.TxTokenData.PropertyID]; !ok {
					Logger.log.Info("Store Cross Shard Custom if It's not existed in DB", customTokenTx.TxTokenData.PropertyID, customTokenTx.TxTokenData.PropertySymbol, customTokenTx.TxTokenData.PropertyName)
					err = blockchain.config.DataBase.StoreCustomToken(&customTokenTx.TxTokenData.PropertyID, customTokenTx.Hash()[:])
				}*/
			}
		case transaction.CustomTokenTransfer:
			{
				Logger.log.Info("Transfer custom token %+v", customTokenTx)
			}
		}
		// save tx which relate to custom token
		// Reject Double spend UTXO before enter this state
		//fmt.Printf("StoreCustomTokenPaymentAddresstHistory/CustomTokenTx: \n VIN %+v VOUT %+v \n", customTokenTx.TxTokenData.Vins, customTokenTx.TxTokenData.Vouts)
		Logger.log.Info("Store Custom Token History")
		err = blockchain.StoreCustomTokenPaymentAddresstHistory(customTokenTx, block.Header.ShardID)
		if err != nil {
			// Skip double spend
			return err
		}
		err = blockchain.config.DataBase.StoreCustomTokenTx(&customTokenTx.TxTokenData.PropertyID, block.Header.ShardID, block.Header.Height, indexTx, customTokenTx.Hash()[:])
		if err != nil {
			return err
		}

		// replace 1000 with proper value for snapshot
		if block.Header.Height%1000 == 0 {
			// list of unreward-utxo
			blockchain.config.customTokenRewardSnapshot, err = blockchain.config.DataBase.GetCustomTokenPaymentAddressesBalance(&customTokenTx.TxTokenData.PropertyID)
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}

	// check privacy custom token
	for indexTx, privacyCustomTokenSubView := range view.privacyCustomTokenViewPoint {
		privacyCustomTokenTx := view.privacyCustomTokenTxs[indexTx]
		switch privacyCustomTokenTx.TxTokenPrivacyData.Type {
		case transaction.CustomTokenInit:
			{
				Logger.log.Info("Store custom token when it is issued", privacyCustomTokenTx.TxTokenPrivacyData.PropertyID, privacyCustomTokenTx.TxTokenPrivacyData.PropertySymbol, privacyCustomTokenTx.TxTokenPrivacyData.PropertyName)
				err = blockchain.config.DataBase.StorePrivacyCustomToken(&privacyCustomTokenTx.TxTokenPrivacyData.PropertyID, privacyCustomTokenTx.Hash()[:])
				if err != nil {
					return err
				}
			}
		case transaction.CustomTokenTransfer:
			{
				Logger.log.Info("Transfer custom token %+v", privacyCustomTokenTx)
			}
		}
		err = blockchain.config.DataBase.StorePrivacyCustomTokenTx(&privacyCustomTokenTx.TxTokenPrivacyData.PropertyID, block.Header.ShardID, block.Header.Height, indexTx, privacyCustomTokenTx.Hash()[:])
		if err != nil {
			return err
		}

		err = blockchain.StoreSerialNumbersFromTxViewPoint(*privacyCustomTokenSubView)
		if err != nil {
			return err
		}

		err = blockchain.StoreCommitmentsFromTxViewPoint(*privacyCustomTokenSubView, block.Header.ShardID)
		if err != nil {
			return err
		}

		err = blockchain.StoreSNDerivatorsFromTxViewPoint(*privacyCustomTokenSubView, block.Header.ShardID)
		if err != nil {
			return err
		}
	}

	// Update the list nullifiers and commitment, snd set using the state of the used tx view point. This
	// entails adding the new
	// ones created by the block.
	err = blockchain.StoreSerialNumbersFromTxViewPoint(*view)
	if err != nil {
		return err
	}

	err = blockchain.StoreCommitmentsFromTxViewPoint(*view, block.Header.ShardID)
	if err != nil {
		return err
	}

	err = blockchain.StoreSNDerivatorsFromTxViewPoint(*view, block.Header.ShardID)
	if err != nil {
		return err
	}

	return nil
}

func (blockchain *BlockChain) CreateAndSaveCrossTransactionCoinViewPointFromBlock(block *ShardBlock) error {
	// Fetch data from block into tx View point
	view := NewTxViewPoint(block.Header.ShardID)

	err := view.fetchCrossTransactionViewPointFromBlock(blockchain.config.DataBase, block)
	if err != nil {
		Logger.log.Error("CreateAndSaveCrossTransactionCoinViewPointFromBlock", err)
	}
	for _, privacyCustomTokenSubView := range view.privacyCustomTokenViewPoint {
		// 0xsirrush updated: check existed tokenID
		tokenID := privacyCustomTokenSubView.tokenID
		existed := blockchain.PrivacyCustomTokenIDExisted(tokenID)
		if !existed {
			existedCrossShard := blockchain.PrivacyCustomTokenIDCrossShardExisted(tokenID)
			if !existedCrossShard {
				Logger.log.Info("Store custom token when it is issued ", tokenID, privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertyName, privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertySymbol, privacyCustomTokenSubView.privacyCustomTokenMetadata.Amount, privacyCustomTokenSubView.privacyCustomTokenMetadata.Mintable)
				tokenDataBytes, _ := json.Marshal(privacyCustomTokenSubView.privacyCustomTokenMetadata)

				// crossShardTokenPrivacyMetaData := CrossShardTokenPrivacyMetaData{}
				// json.Unmarshal(tokenDataBytes, &crossShardTokenPrivacyMetaData)
				// fmt.Println("New Token CrossShardTokenPrivacyMetaData", crossShardTokenPrivacyMetaDatla)

				if err := blockchain.config.DataBase.StorePrivacyCustomTokenCrossShard(tokenID, tokenDataBytes); err != nil {
					return err
				}
			}
		}
		/*listCustomTokens, listCustomTokenCrossShard, err := blockchain.ListPrivacyCustomToken()
		if err != nil {
			return nil
		}
		tokenID := privacyCustomTokenSubView.tokenID
		if _, ok := listCustomTokens[*tokenID]; !ok {
			if _, ok := listCustomTokenCrossShard[*tokenID]; !ok {
				Logger.log.Info("Store custom token when it is issued ", tokenID, privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertyName, privacyCustomTokenSubView.privacyCustomTokenMetadata.PropertySymbol, privacyCustomTokenSubView.privacyCustomTokenMetadata.Amount, privacyCustomTokenSubView.privacyCustomTokenMetadata.Mintable)
				tokenDataBytes, _ := json.Marshal(privacyCustomTokenSubView.privacyCustomTokenMetadata)

				// crossShardTokenPrivacyMetaData := CrossShardTokenPrivacyMetaData{}
				// json.Unmarshal(tokenDataBytes, &crossShardTokenPrivacyMetaData)
				// fmt.Println("New Token CrossShardTokenPrivacyMetaData", crossShardTokenPrivacyMetaData)

				if err := blockchain.config.DataBase.StorePrivacyCustomTokenCrossShard(tokenID, tokenDataBytes); err != nil {
					return err
				}
			}
		}*/
		// Store both commitment and outcoin
		err = blockchain.StoreCommitmentsFromTxViewPoint(*privacyCustomTokenSubView, block.Header.ShardID)
		if err != nil {
			return err
		}
		// store snd
		err = blockchain.StoreSNDerivatorsFromTxViewPoint(*privacyCustomTokenSubView, block.Header.ShardID)
		if err != nil {
			return err
		}
	}

	// Update the list nullifiers and commitment, snd set using the state of the used tx view point. This
	// entails adding the new
	// ones created by the block.
	err = blockchain.StoreCommitmentsFromTxViewPoint(*view, block.Header.ShardID)
	if err != nil {
		return err
	}

	err = blockchain.StoreSNDerivatorsFromTxViewPoint(*view, block.Header.ShardID)
	if err != nil {
		return err
	}

	return nil
}

/*
// 	KeyWallet: token-paymentAddress  -[-]-  {tokenId}  -[-]-  {paymentAddress}  -[-]-  {txHash}  -[-]-  {voutIndex}
//   H: value-spent/unspent
*/
func (blockchain *BlockChain) StoreCustomTokenPaymentAddresstHistory(customTokenTx *transaction.TxCustomToken, shardID byte) error {
	Splitter := lvdb.Splitter
	TokenPaymentAddressPrefix := lvdb.TokenPaymentAddressPrefix
	unspent := lvdb.Unspent
	spent := lvdb.Spent
	unreward := lvdb.Unreward

	tokenKey := TokenPaymentAddressPrefix
	tokenKey = append(tokenKey, Splitter...)
	tokenKey = append(tokenKey, []byte((customTokenTx.TxTokenData.PropertyID).String())...)
	for _, vin := range customTokenTx.TxTokenData.Vins {
		paymentAddressBytes := base58.Base58Check{}.Encode(vin.PaymentAddress.Bytes(), 0x00)
		utxoHash := []byte(vin.TxCustomTokenID.String())
		voutIndex := vin.VoutIndex
		paymentAddressKey := tokenKey
		paymentAddressKey = append(paymentAddressKey, Splitter...)
		paymentAddressKey = append(paymentAddressKey, paymentAddressBytes...)
		paymentAddressKey = append(paymentAddressKey, Splitter...)
		paymentAddressKey = append(paymentAddressKey, utxoHash[:]...)
		paymentAddressKey = append(paymentAddressKey, Splitter...)
		paymentAddressKey = append(paymentAddressKey, common.Int32ToBytes(int32(voutIndex))...)
		_, err := blockchain.config.DataBase.HasValue(paymentAddressKey)
		if err != nil {
			return err
		}
		value, err := blockchain.config.DataBase.Get(paymentAddressKey)
		if err != nil {
			return err
		}
		// old value: {value}-unspent-unreward/reward
		values := strings.Split(string(value), string(Splitter))
		if strings.Compare(values[1], string(unspent)) != 0 {
			return errors.New("Double Spend Detected")
		}
		// new value: {value}-spent-unreward/reward
		newValues := values[0] + string(Splitter) + string(spent) + string(Splitter) + values[2]
		if err := blockchain.config.DataBase.Put(paymentAddressKey, []byte(newValues)); err != nil {
			return err
		}
	}
	for index, vout := range customTokenTx.TxTokenData.Vouts {
		// check vout by type and receiver
		txCustomTokenType := customTokenTx.TxTokenData.Type
		if txCustomTokenType == transaction.CustomTokenInit || txCustomTokenType == transaction.CustomTokenTransfer {
			// check receiver's shard and current shard ID
			shardIDOfReceiver := common.GetShardIDFromLastByte(vout.PaymentAddress.Pk[len(vout.PaymentAddress.Pk)-1])
			if shardIDOfReceiver != shardID {
				continue
			}
		} else if txCustomTokenType == transaction.CustomTokenCrossShard {
			shardIDOfReceiver := common.GetShardIDFromLastByte(vout.PaymentAddress.Pk[len(vout.PaymentAddress.Pk)-1])
			if shardIDOfReceiver != shardID {
				continue
			}
		}
		paymentAddressBytes := base58.Base58Check{}.Encode(vout.PaymentAddress.Bytes(), 0x00)
		utxoHash := []byte(customTokenTx.Hash().String())
		voutIndex := index
		value := vout.Value
		paymentAddressKey := tokenKey
		paymentAddressKey = append(paymentAddressKey, Splitter...)
		paymentAddressKey = append(paymentAddressKey, paymentAddressBytes...)
		paymentAddressKey = append(paymentAddressKey, Splitter...)
		paymentAddressKey = append(paymentAddressKey, utxoHash[:]...)
		paymentAddressKey = append(paymentAddressKey, Splitter...)
		paymentAddressKey = append(paymentAddressKey, common.Int32ToBytes(int32(voutIndex))...)
		ok, err := blockchain.config.DataBase.HasValue(paymentAddressKey)
		// Vout already exist
		if ok {
			return errors.New("UTXO already exist")
		}
		if err != nil {
			return err
		}
		// init value: {value}-unspent-unreward
		paymentAddressValue := strconv.Itoa(int(value)) + string(Splitter) + string(unspent) + string(Splitter) + string(unreward)
		if err := blockchain.config.DataBase.Put(paymentAddressKey, []byte(paymentAddressValue)); err != nil {
			return err
		}
		fmt.Printf("STORE UTXO FOR CUSTOM TOKEN: tokenID %+v \n paymentAddress %+v \n txHash %+v, voutIndex %+v, value %+v \n", (customTokenTx.TxTokenData.PropertyID).String(), vout.PaymentAddress, customTokenTx.Hash(), voutIndex, value)
	}
	return nil
}

// DecryptTxByKey - process outputcoin to get outputcoin data which relate to keyset
func (blockchain *BlockChain) DecryptOutputCoinByKey(outCoinTemp *privacy.OutputCoin, keySet *cashec.KeySet, shardID byte, tokenID *common.Hash) *privacy.OutputCoin {
	/*
		- Param keyset - (priv-key, payment-address, readonlykey)
		in case priv-key: return unspent outputcoin tx
		in case readonly-key: return all outputcoin tx with amount value
		in case payment-address: return all outputcoin tx with no amount value
	*/
	pubkeyCompress := outCoinTemp.CoinDetails.PublicKey.Compress()
	if bytes.Equal(pubkeyCompress, keySet.PaymentAddress.Pk[:]) {
		result := &privacy.OutputCoin{
			CoinDetails:          outCoinTemp.CoinDetails,
			CoinDetailsEncrypted: outCoinTemp.CoinDetailsEncrypted,
		}
		if result.CoinDetailsEncrypted != nil {
			if len(keySet.PrivateKey) > 0 || len(keySet.ReadonlyKey.Rk) > 0 {
				// try to decrypt to get more data
				err := result.Decrypt(keySet.ReadonlyKey)
				if err == nil {
					result.CoinDetails = outCoinTemp.CoinDetails
				}
			}
		}
		if len(keySet.PrivateKey) > 0 {
			// check spent with private-key
			result.CoinDetails.SerialNumber = privacy.PedCom.G[privacy.SK].Derive(new(big.Int).SetBytes(keySet.PrivateKey),
				result.CoinDetails.SNDerivator)
			ok, err := blockchain.config.DataBase.HasSerialNumber(tokenID, result.CoinDetails.SerialNumber.Compress(), shardID)
			if ok || err != nil {
				return nil
			}
		}
		return result
	}
	return nil
}

/*
GetListOutputCoinsByKeyset - Read all blocks to get txs(not action tx) which can be decrypt by readonly secret key.
With private-key, we can check unspent tx by check nullifiers from database
- Param #1: keyset - (priv-key, payment-address, readonlykey)
in case priv-key: return unspent outputcoin tx
in case readonly-key: return all outputcoin tx with amount value
in case payment-address: return all outputcoin tx with no amount value
- Param #2: coinType - which type of joinsplitdesc(COIN or BOND)
*/
func (blockchain *BlockChain) GetListOutputCoinsByKeyset(keyset *cashec.KeySet, shardID byte, tokenID *common.Hash) ([]*privacy.OutputCoin, error) {
	// lock chain
	blockchain.BestState.Shard[shardID].lock.Lock()
	defer blockchain.BestState.Shard[shardID].lock.Unlock()

	outCointsInBytes, err := blockchain.config.DataBase.GetOutcoinsByPubkey(tokenID, keyset.PaymentAddress.Pk[:], shardID)
	if err != nil {
		return nil, err
	}
	// convert from []byte to object
	outCoints := make([]*privacy.OutputCoin, 0)
	for _, item := range outCointsInBytes {
		outcoin := &privacy.OutputCoin{}
		outcoin.Init()
		outcoin.SetBytes(item)
		outCoints = append(outCoints, outcoin)
	}

	// loop on all outputcoin to decrypt data
	results := make([]*privacy.OutputCoin, 0)
	for _, out := range outCoints {
		pubkeyCompress := out.CoinDetails.PublicKey.Compress()
		if bytes.Equal(pubkeyCompress, keyset.PaymentAddress.Pk[:]) {
			out = blockchain.DecryptOutputCoinByKey(out, keyset, shardID, tokenID)
			if out == nil {
				continue
			} else {
				results = append(results, out)
			}
		}
	}
	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetUnspentTxCustomTokenVout - return all unspent tx custom token out of sender
func (blockchain *BlockChain) GetUnspentTxCustomTokenVout(receiverKeyset cashec.KeySet, tokenID *common.Hash) ([]transaction.TxTokenVout, error) {
	data, err := blockchain.config.DataBase.GetCustomTokenPaymentAddressUTXO(tokenID, receiverKeyset.PaymentAddress.Bytes())
	fmt.Println(data)
	if err != nil {
		return nil, err
	}
	splitter := []byte("-[-]-")
	unspent := []byte("unspent")
	voutList := []transaction.TxTokenVout{}
	for key, value := range data {
		keys := strings.Split(key, string(splitter))
		values := strings.Split(value, string(splitter))
		// values: [amount-value, spent/unspent]
		// get unspent transaction output
		if strings.Compare(values[1], string(unspent)) == 0 {
			vout := transaction.TxTokenVout{}
			vout.PaymentAddress = receiverKeyset.PaymentAddress
			txHash, err := common.Hash{}.NewHashFromStr(string(keys[3]))
			if err != nil {
				return nil, err
			}
			vout.SetTxCustomTokenID(*txHash)
			voutIndexByte := []byte(keys[4])
			voutIndex := common.BytesToInt32(voutIndexByte)
			vout.SetIndex(int(voutIndex))
			value, err := strconv.Atoi(values[0])
			if err != nil {
				return nil, err
			}
			vout.Value = uint64(value)
			Logger.log.Info("GetCustomTokenPaymentAddressUTXO VOUT", vout)
			voutList = append(voutList, vout)
		}
	}
	return voutList, nil
}

// GetTransactionByHash - retrieve tx from txId(txHash)
func (blockchain *BlockChain) GetTransactionByHash(txHash *common.Hash) (byte, *common.Hash, int, metadata.Transaction, error) {
	blockHash, index, err := blockchain.config.DataBase.GetTransactionIndexById(txHash)
	if err != nil {
		abc := NewBlockChainError(UnExpectedError, err)
		Logger.log.Error(abc)
		return byte(255), nil, -1, nil, abc
	}
	block, _, err1 := blockchain.GetShardBlockByHash(blockHash)
	if err1 != nil {
		Logger.log.Errorf("ERROR", err1, "NO Transaction in block with hash &+v", blockHash, "and index", index, "contains", block.Body.Transactions[index])
		return byte(255), nil, -1, nil, NewBlockChainError(UnExpectedError, err1)
	}
	//Logger.log.Infof("Transaction in block with hash &+v", blockHash, "and index", index, "contains", block.Transactions[index])
	return block.Header.ShardID, blockHash, index, block.Body.Transactions[index], nil
}

// Check Custom token ID is existed
func (blockchain *BlockChain) CustomTokenIDExisted(tokenID *common.Hash) bool {
	return blockchain.config.DataBase.CustomTokenIDExisted(tokenID)
}

// Check Privacy Custom token ID is existed
func (blockchain *BlockChain) PrivacyCustomTokenIDExisted(tokenID *common.Hash) bool {
	return blockchain.config.DataBase.PrivacyCustomTokenIDExisted(tokenID)
}

func (blockchain *BlockChain) PrivacyCustomTokenIDCrossShardExisted(tokenID *common.Hash) bool {
	return blockchain.config.DataBase.PrivacyCustomTokenIDCrossShardExisted(tokenID)
}

// ListCustomToken - return all custom token which existed in network
func (blockchain *BlockChain) ListCustomToken() (map[common.Hash]transaction.TxCustomToken, error) {
	data, err := blockchain.config.DataBase.ListCustomToken()
	if err != nil {
		return nil, err
	}
	result := make(map[common.Hash]transaction.TxCustomToken)
	for _, txData := range data {
		hash := common.Hash{}
		hash.SetBytes(txData)
		_, blockHash, index, tx, err := blockchain.GetTransactionByHash(&hash)
		_ = blockHash
		_ = index
		if err != nil {
			return nil, NewBlockChainError(UnExpectedError, err)
		}
		txCustomToken := tx.(*transaction.TxCustomToken)
		result[txCustomToken.TxTokenData.PropertyID] = *txCustomToken
	}
	return result, nil
}

// ListCustomToken - return all custom token which existed in network
func (blockchain *BlockChain) ListPrivacyCustomToken() (map[common.Hash]transaction.TxCustomTokenPrivacy, map[common.Hash]CrossShardTokenPrivacyMetaData, error) {
	data, err := blockchain.config.DataBase.ListPrivacyCustomToken()
	if err != nil {
		return nil, nil, err
	}
	crossShardData, err := blockchain.config.DataBase.ListPrivacyCustomTokenCrossShard()
	if err != nil {
		return nil, nil, err
	}
	result := make(map[common.Hash]transaction.TxCustomTokenPrivacy)
	for _, txData := range data {
		hash := common.Hash{}
		hash.SetBytes(txData)
		_, blockHash, index, tx, err := blockchain.GetTransactionByHash(&hash)
		_ = blockHash
		_ = index
		if err != nil {
			return nil, nil, err
		}
		txPrivacyCustomToken := tx.(*transaction.TxCustomTokenPrivacy)
		result[txPrivacyCustomToken.TxTokenPrivacyData.PropertyID] = *txPrivacyCustomToken
	}
	resultCrossShard := make(map[common.Hash]CrossShardTokenPrivacyMetaData)
	for _, tokenData := range crossShardData {
		crossShardTokenPrivacyMetaData := CrossShardTokenPrivacyMetaData{}
		err = json.Unmarshal(tokenData, &crossShardTokenPrivacyMetaData)
		if err != nil {
			return nil, nil, err
		}
		resultCrossShard[crossShardTokenPrivacyMetaData.TokenID] = crossShardTokenPrivacyMetaData
	}
	return result, resultCrossShard, nil
}

// GetCustomTokenTxsHash - return list hash of tx which relate to custom token
func (blockchain *BlockChain) GetCustomTokenTxsHash(tokenID *common.Hash) ([]common.Hash, error) {
	txHashesInByte, err := blockchain.config.DataBase.CustomTokenTxs(tokenID)
	if err != nil {
		return nil, err
	}
	result := []common.Hash{}
	for _, temp := range txHashesInByte {
		result = append(result, *temp)
	}
	return result, nil
}

// GetPrivacyCustomTokenTxsHash - return list hash of tx which relate to custom token
func (blockchain *BlockChain) GetPrivacyCustomTokenTxsHash(tokenID *common.Hash) ([]common.Hash, error) {
	txHashesInByte, err := blockchain.config.DataBase.PrivacyCustomTokenTxs(tokenID)
	if err != nil {
		return nil, err
	}
	result := []common.Hash{}
	for _, temp := range txHashesInByte {
		result = append(result, *temp)
	}
	return result, nil
}

// GetListTokenHolders - return list paymentaddress (in hexstring) of someone who hold custom token in network
func (blockchain *BlockChain) GetListTokenHolders(tokenID *common.Hash) (map[string]uint64, error) {
	result, err := blockchain.config.DataBase.GetCustomTokenPaymentAddressesBalance(tokenID)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (blockchain *BlockChain) GetCustomTokenRewardSnapshot() map[string]uint64 {
	return blockchain.config.customTokenRewardSnapshot
}

func (self *BlockChain) GetCurrentBeaconBlockHeight(shardID byte) uint64 {
	return self.BestState.Beacon.BestBlock.Header.Height
}

func (blockchain BlockChain) RandomCommitmentsProcess(usableInputCoins []*privacy.InputCoin, randNum int, shardID byte, tokenID *common.Hash) (commitmentIndexs []uint64, myCommitmentIndexs []uint64, commitments [][]byte) {
	return transaction.RandomCommitmentsProcess(usableInputCoins, randNum, blockchain.config.DataBase, shardID, tokenID)
}

func (blockchain BlockChain) CheckSNDerivatorExistence(tokenID *common.Hash, snd *big.Int, shardID byte) (bool, error) {
	return transaction.CheckSNDerivatorExistence(tokenID, snd, shardID, blockchain.config.DataBase)
}

// GetFeePerKbTx - return fee (per kb of tx)
func (blockchain BlockChain) GetFeePerKbTx() uint64 {
	return 0
}

// GetRecentTransactions - find all recent history txs which are created by user
// by number of block, maximum is 100 newest blocks
func (blockchain *BlockChain) GetRecentTransactions(numBlock uint64, key *privacy.ViewingKey, shardID byte) (map[string]metadata.Transaction, error) {
	if numBlock > 100 { // maximum is 100
		numBlock = 100
	}
	// var err error
	result := make(map[string]metadata.Transaction)
	// bestBlock := blockchain.BestState[shardID].BestBlock
	// for {
	// 	for _, tx := range bestBlock.Transactions {
	// 		info := tx.GetInfo()
	// 		if info == nil {
	// 			continue
	// 		}
	// 		// info of tx with contain encrypted pubkey of creator in 1st 64bytes
	// 		lenInfo := 66
	// 		if len(info) < lenInfo {
	// 			continue
	// 		}
	// 		// decrypt to get pubkey data from info
	// 		pubkeyData, err1 := privacy.ElGamalDecrypt(key.Rk[:], info[0:lenInfo])
	// 		if err1 != nil {
	// 			continue
	// 		}
	// 		// compare to check pubkey
	// 		if !bytes.Equal(pubkeyData.Compress(), key.Pk[:]) {
	// 			continue
	// 		}
	// 		result[tx.Hash().String()] = tx
	// 	}
	// 	numBlock--
	// 	if numBlock == 0 {
	// 		break
	// 	}
	// 	bestBlock, err = blockchain.GetBlockByBlockHash(&bestBlock.Header.PrevBlockHash)
	// 	if err != nil {
	// 		break
	// 	}
	// }
	return result, nil
}

func (blockchain *BlockChain) SetReadyState(shard bool, shardID byte, ready bool) {
	// fmt.Println("SetReadyState", shard, shardID, ready)
	blockchain.syncStatus.IsReady.Lock()
	defer blockchain.syncStatus.IsReady.Unlock()
	if shard {
		blockchain.syncStatus.IsReady.Shards[shardID] = ready
	} else {
		blockchain.syncStatus.IsReady.Beacon = ready
		if ready {
			fmt.Println("blockchain is ready")
		}
	}
}

func (blockchain *BlockChain) IsReady(shard bool, shardID byte) bool {
	blockchain.syncStatus.IsReady.Lock()
	defer blockchain.syncStatus.IsReady.Unlock()
	if shard {
		if _, ok := blockchain.syncStatus.IsReady.Shards[shardID]; !ok {
			return false
		}
		return blockchain.syncStatus.IsReady.Shards[shardID]
	}
	return blockchain.syncStatus.IsReady.Beacon
}
