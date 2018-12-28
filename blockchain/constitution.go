package blockchain

// import (
// 	"github.com/ninjadotorg/constant/blockchain/params"
// 	"github.com/ninjadotorg/constant/common"
// 	"github.com/ninjadotorg/constant/metadata"
// 	"github.com/ninjadotorg/constant/privacy"
// 	"github.com/ninjadotorg/constant/transaction"
// )

// type ConstitutionInfo struct {
// 	StartedBlockHeight uint32
// 	ExecuteDuration    uint32
// 	AcceptProposalTXID common.Hash
// }

// func NewConstitutionInfo(startedBlockHeight uint32, executeDuration uint32, proposalTXID common.Hash) *ConstitutionInfo {
// 	return &ConstitutionInfo{
// 		StartedBlockHeight: startedBlockHeight,
// 		ExecuteDuration:    executeDuration,
// 		AcceptProposalTXID: proposalTXID,
// 	}
// }

// type GOVConstitution struct {
// 	ConstitutionInfo
// 	CurrentGOVNationalWelfare int32
// 	GOVParams                 params.GOVParams
// }

// func NewGOVConstitution(constitutionInfo *ConstitutionInfo, currentGOVNationalWelfare int32, GOVParams *params.GOVParams) *GOVConstitution {
// 	return &GOVConstitution{
// 		ConstitutionInfo:          *constitutionInfo,
// 		CurrentGOVNationalWelfare: currentGOVNationalWelfare,
// 		GOVParams:                 *GOVParams,
// 	}
// }

// // type DCBConstitutionHelper struct{}
// // type GOVConstitutionHelper struct{}

// // func (DCBConstitutionHelper) GetEndedBlockHeight(blockgen *BlkTmplGenerator, shardID byte) uint32 {
// // 	BestBlock := blockgen.chain.BestState[shardID].BestBlock
// // 	lastDCBConstitution := BestBlock.Header.DCBConstitution
// // 	return lastDCBConstitution.StartedBlockHeight + lastDCBConstitution.ExecuteDuration
// // }

// func NewDCBConstitution(constitutionInfo *ConstitutionInfo, currentDCBNationalWelfare int32, DCBParams *params.DCBParams) *DCBConstitution {
// 	return &DCBConstitution{
// 		ConstitutionInfo:          *constitutionInfo,
// 		CurrentDCBNationalWelfare: currentDCBNationalWelfare,
// 		DCBParams:                 *DCBParams,
// 	}
// }

// type DCBConstitutionHelper struct{}
// type GOVConstitutionHelper struct{}

// func (DCBConstitutionHelper) GetConstitutionEndedBlockHeight(blockgen *BlkTmplGenerator, chainID byte) uint32 {
// 	BestBlock := blockgen.chain.BestState[chainID].BestBlock
// 	lastDCBConstitution := BestBlock.Header.DCBConstitution
// 	return lastDCBConstitution.StartedBlockHeight + lastDCBConstitution.ExecuteDuration
// }

// func (GOVConstitutionHelper) GetConstitutionEndedBlockHeight(blockgen *BlkTmplGenerator, chainID byte) uint32 {
// 	BestBlock := blockgen.chain.BestState[chainID].BestBlock
// 	lastGOVConstitution := BestBlock.Header.GOVConstitution
// 	return lastGOVConstitution.StartedBlockHeight + lastGOVConstitution.ExecuteDuration
// }

// // // func (DCBConstitutionHelper) GetAmountVoteToken(tx metadata.Transaction) uint64 {
// // // 	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
// // // }

// // // func (GOVConstitutionHelper) GetStartedNormalVote(blockgen *BlkTmplGenerator, shardID byte) int32 {
// // // 	BestBlock := blockgen.chain.BestState[shardID].BestBlock
// // // 	lastGOVConstitution := BestBlock.Header.GOVConstitution
// // // 	return lastGOVConstitution.StartedBlockHeight - common.EncryptionPhaseDuration
// // // }

// // func (GOVConstitutionHelper) GetStartedNormalVote(blockgen *BlkTmplGenerator, shardID byte) uint32 {
// // 	BestBlock := blockgen.chain.BestState[shardID].BestBlock
// // 	lastGOVConstitution := BestBlock.Header.GOVConstitution
// // 	return lastGOVConstitution.StartedBlockHeight - common.EncryptionPhaseDuration
// // }

// // // func (GOVConstitutionHelper) CheckVotingProposalType(tx metadata.Transaction) bool {
// // // 	return tx.GetMetadataType() == metadata.VoteGOVProposalMeta
// // // }

// // // func (GOVConstitutionHelper) GetAmountVoteToken(tx metadata.Transaction) uint64 {
// // // 	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
// // // }

// // // func (DCBConstitutionHelper) TxAcceptProposal(originTx metadata.Transaction) metadata.Transaction {
// // // 	acceptTx := transaction.Tx{
// // // 		Metadata: &metadata.AcceptDCBProposalMetadata{
// // // 			DCBProposalTXID: *originTx.Hash(),
// // // 		},
// // // 	}
// // // 	return &acceptTx
// // // }

// // func (DCBConstitutionHelper) TxAcceptProposal(txId *common.Hash) metadata.Transaction {
// // 	acceptTx := transaction.Tx{
// // 		Metadata: &metadata.AcceptDCBProposalMetadata{
// // 			DCBProposalTXID: *txId,
// // 		},
// // 	}
// // 	return &acceptTx
// // }

// // func (GOVConstitutionHelper) TxAcceptProposal(txId *common.Hash) metadata.Transaction {
// // 	acceptTx := transaction.Tx{
// // 		Metadata: &metadata.AcceptGOVProposalMetadata{
// // 			GOVProposalTXID: *txId,
// // 		},
// // 	}
// // 	return &acceptTx
// // }

// func (DCBConstitutionHelper) TxAcceptProposal(txId *common.Hash, voter metadata.Voter) metadata.Transaction {
// 	acceptTx := transaction.Tx{
// 		Metadata: metadata.NewAcceptDCBProposalMetadata(*txId, voter),
// 	}
// 	return &acceptTx
// }

// func (GOVConstitutionHelper) TxAcceptProposal(txId *common.Hash, voter metadata.Voter) metadata.Transaction {
// 	acceptTx := transaction.Tx{
// 		Metadata: metadata.NewAcceptGOVProposalMetadata(*txId, voter),
// 	}
// 	return &acceptTx
// }

// func (DCBConstitutionHelper) GetLowerCaseBoardType() string {
// 	return "dcb"
// }

// func (GOVConstitutionHelper) GetLowerCaseBoardType() string {
// 	return "gov"
// }

// func (DCBConstitutionHelper) CreatePunishDecryptTx(pubKey []byte) metadata.Metadata {
// 	return metadata.NewPunishDCBDecryptMetadata(pubKey)
// }

// func (GOVConstitutionHelper) CreatePunishDecryptTx(pubKey []byte) metadata.Metadata {
// 	return metadata.NewPunishGOVDecryptMetadata(pubKey)
// }

// func (DCBConstitutionHelper) GetSealerPubKey(tx metadata.Transaction) [][]byte {
// 	meta := tx.GetMetadata().(*metadata.SealedLv3DCBBallotMetadata)
// 	return meta.SealedDCBBallot.LockerPubKeys
// }

// func (GOVConstitutionHelper) GetSealerPubKey(tx metadata.Transaction) [][]byte {
// 	meta := tx.GetMetadata().(*metadata.SealedLv3GOVBallotMetadata)
// 	return meta.SealedGOVBallot.LockerPubKeys
// }

// func (DCBConstitutionHelper) NewTxRewardProposalSubmitter(blockgen *BlkTmplGenerator, receiverAddress *privacy.PaymentAddress, minerPrivateKey *privacy.SpendingKey) (metadata.Transaction, error) {
// 	tx := transaction.Tx{}
// 	err := tx.InitTxSalary(common.RewardProposalSubmitter, receiverAddress, minerPrivateKey, blockgen.chain.config.DataBase)
// 	if err != nil {
// 		return nil, err
// 	}
// 	meta := metadata.NewRewardDCBProposalSubmitterMetadata()
// 	tx.SetMetadata(meta)
// 	return &tx, nil
// }

// func (GOVConstitutionHelper) NewTxRewardProposalSubmitter(blockgen *BlkTmplGenerator, receiverAddress *privacy.PaymentAddress, minerPrivateKey *privacy.SpendingKey) (metadata.Transaction, error) {
// 	tx := transaction.Tx{}
// 	err := tx.InitTxSalary(common.RewardProposalSubmitter, receiverAddress, minerPrivateKey, blockgen.chain.config.DataBase)
// 	if err != nil {
// 		return nil, err
// 	}
// 	meta := metadata.NewRewardGOVProposalSubmitterMetadata()
// 	tx.SetMetadata(meta)
// 	return &tx, nil
// }

// func (DCBConstitutionHelper) GetPaymentAddressFromSubmitProposalMetadata(tx metadata.Transaction) *privacy.PaymentAddress {
// 	meta := tx.GetMetadata().(*metadata.SubmitDCBProposalMetadata)
// 	return &meta.PaymentAddress
// }
// func (GOVConstitutionHelper) GetPaymentAddressFromSubmitProposalMetadata(tx metadata.Transaction) *privacy.PaymentAddress {
// 	meta := tx.GetMetadata().(*metadata.SubmitGOVProposalMetadata)
// 	return &meta.PaymentAddress
// }

// func (DCBConstitutionHelper) GetPubKeyVoter(blockgen *BlkTmplGenerator, chainID byte) ([]byte, error) {
// 	bestBlock := blockgen.chain.BestState[chainID].BestBlock
// 	_, _, _, tx, _ := blockgen.chain.GetTransactionByHash(&bestBlock.Header.DCBConstitution.AcceptProposalTXID)
// 	meta := tx.GetMetadata().(*metadata.AcceptDCBProposalMetadata)
// 	return meta.Voter.PubKey, nil
// }
// func (GOVConstitutionHelper) GetPubKeyVoter(blockgen *BlkTmplGenerator, chainID byte) ([]byte, error) {
// 	bestBlock := blockgen.chain.BestState[chainID].BestBlock
// 	_, _, _, tx, _ := blockgen.chain.GetTransactionByHash(&bestBlock.Header.GOVConstitution.AcceptProposalTXID)
// 	meta := tx.GetMetadata().(*metadata.AcceptGOVProposalMetadata)
// 	return meta.Voter.PubKey, nil
// }

// func (DCBConstitutionHelper) GetPrizeProposal() uint32 {
// 	return uint32(common.Maxint32(GetOracleDCBNationalWelfare(), int32(0)))
// }

// func (GOVConstitutionHelper) GetPrizeProposal() uint32 {
// 	return uint32(common.Maxint32(GetOracleGOVNationalWelfare(), int32(0)))
// }

// func (GOVConstitutionHelper) GetPrizeProposal() uint32 {
// 	return uint32(common.Maxint32(GetOracleGOVNationalWelfare(), int32(0)))
// }

// func (DCBConstitutionHelper) GetTopMostVoteGovernor(blockgen *BlkTmplGenerator) (database.CandidateList, error) {
// 	return blockgen.chain.config.DataBase.GetTopMostVoteDCBGovernor(common.NumberOfDCBGovernors)
// }
// func (GOVConstitutionHelper) GetTopMostVoteGovernor(blockgen *BlkTmplGenerator) (database.CandidateList, error) {
// 	return blockgen.chain.config.DataBase.GetTopMostVoteGOVGovernor(common.NumberOfGOVGovernors)
// }
