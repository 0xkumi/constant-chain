package rpcserver

// rpc cmd method
const (
	StartProfiling = "startprofiling"
	StopProfiling  = "stopprofiling"

	GetNetworkInfo     = "getnetworkinfo"
	GetConnectionCount = "getconnectioncount"
	GetAllPeers        = "getallpeers"

	EstimateFee              = "estimatefee"
	EstimateFeeWithEstimator = "estimatefeewithestimator"

	GetActiveShards    = "getactiveshards"
	GetMaxShardsNumber = "getmaxshardsnumber"

	GetMiningInfo                 = "getmininginfo"
	GetRawMempool                 = "getrawmempool"
	GetNumberOfTxsInMempool       = "getnumberoftxsinmempool"
	GetMempoolEntry               = "getmempoolentry"
	GetBeaconPoolState            = "getbeaconpoolstate"
	GetShardPoolState             = "getshardpoolstate"
	GetShardPoolLatestValidHeight = "getshardpoollatestvalidheight"
	GetShardToBeaconPoolState     = "getshardtobeaconpoolstate"
	GetCrossShardPoolState        = "getcrossshardpoolstate"
	GetNextCrossShard             = "getnextcrossshard"
	GetShardToBeaconPoolStateV2   = "getshardtobeaconpoolstatev2"
	GetCrossShardPoolStateV2      = "getcrossshardpoolstatev2"
	GetShardPoolStateV2           = "getshardpoolstatev2"
	GetBeaconPoolStateV2          = "getbeaconpoolstatev2"

	GetBestBlock        = "getbestblock"
	GetBestBlockHash    = "getbestblockhash"
	GetBlocks           = "getblocks"
	RetrieveBlock       = "retrieveblock"
	RetrieveBeaconBlock = "retrievebeaconblock"
	GetBlockChainInfo   = "getblockchaininfo"
	GetBlockCount       = "getblockcount"
	GetBlockHash        = "getblockhash"

	ListOutputCoins                            = "listoutputcoins"
	CreateRawTransaction                       = "createtransaction"
	SendRawTransaction                         = "sendtransaction"
	CreateAndSendTransaction                   = "createandsendtransaction"
	CreateAndSendCustomTokenTransaction        = "createandsendcustomtokentransaction"
	SendRawCustomTokenTransaction              = "sendrawcustomtokentransaction"
	CreateRawCustomTokenTransaction            = "createrawcustomtokentransaction"
	CreateRawPrivacyCustomTokenTransaction     = "createrawprivacycustomtokentransaction"
	SendRawPrivacyCustomTokenTransaction       = "sendrawprivacycustomtokentransaction"
	CreateAndSendPrivacyCustomTokenTransaction = "createandsendprivacycustomtokentransaction"
	GetMempoolInfo                             = "getmempoolinfo"
	GetCandidateList                           = "getcandidatelist"
	GetCommitteeList                           = "getcommitteelist"
	CanPubkeyStake                             = "canpubkeystake"
	GetTotalTransaction                        = "gettotaltransaction"
	GetBlockProducerList                       = "getblockproducer"
	ListUnspentCustomToken                     = "listunspentcustomtoken"
	GetTransactionByHash                       = "gettransactionbyhash"
	ListCustomToken                            = "listcustomtoken"
	ListPrivacyCustomToken                     = "listprivacycustomtoken"
	CustomToken                                = "customtoken"
	PrivacyCustomToken                         = "privacycustomtoken"
	CheckHashValue                             = "checkhashvalue"
	GetListCustomTokenBalance                  = "getlistcustomtokenbalance"
	GetListPrivacyCustomTokenBalance           = "getlistprivacycustomtokenbalance"
	GetBlockHeader                             = "getheader"
	GetCrossShardBlock                         = "getcrossshardblock"
	RandomCommitments                          = "randomcommitments"
	HasSerialNumbers                           = "hasserialnumbers"
	HasSnDerivators                            = "hassnderivators"

	CreateAndSendStakingTransaction = "createandsendstakingtransaction"

	//===========For Testing and Benchmark==============
	GetAndSendTxsFromFile = "getandsendtxsfromfile"
	//==================================================

	GetShardBestState  = "getshardbeststate"
	GetBeaconBestState = "getbeaconbeststate"

	// Wallet rpc cmd
	ListAccounts                       = "listaccounts"
	GetAccount                         = "getaccount"
	GetAddressesByAccount              = "getaddressesbyaccount"
	GetAccountAddress                  = "getaccountaddress"
	DumpPrivkey                        = "dumpprivkey"
	ImportAccount                      = "importaccount"
	RemoveAccount                      = "removeaccount"
	ListUnspentOutputCoins             = "listunspentoutputcoins"
	GetBalance                         = "getbalance"
	GetBalanceByPrivatekey             = "getbalancebyprivatekey"
	GetBalanceByPaymentAddress         = "getbalancebypaymentaddress"
	GetReceivedByAccount               = "getreceivedbyaccount"
	SetTxFee                           = "settxfee"
	GetRecentTransactionsByBlockNumber = "getrecenttransactionsbyblocknumber"

	// multisig for board spending
	CreateSignatureOnCustomTokenTx       = "createsignatureoncustomtokentx"
	GetConstantCirculating               = "getconstantcirculating"
	CreateAndSendTxWithMultiSigsReg      = "createandsendtxwithmultisigsreg"
	CreateAndSendTxWithMultiSigsSpending = "createandsendtxwithmultisigsspending"

	// walletsta
	GetPublicKeyFromPaymentAddress = "getpublickeyfrompaymentaddress"
	DefragmentAccount              = "defragmentaccount"

	GetStackingAmount = "getstackingamount"

	// utils
	HashToIdenticon = "hashtoidenticon"

	CreateIssuingRequest            = "createissuingrequest"
	SendIssuingRequest              = "sendissuingrequest"
	CreateAndSendIssuingRequest     = "createandsendissuingrequest"
	CreateAndSendContractingRequest = "createandsendcontractingrequest"
	GetBridgeTokensAmounts          = "getbridgetokensamounts"
)
