package metadata

import (
	"bytes"
	"encoding/json"

	// "errors"
	"strconv"

	"github.com/constant-money/constant-chain/common"
	"github.com/constant-money/constant-chain/database"
	"github.com/constant-money/constant-chain/privacy"
	"github.com/pkg/errors"
)

type BeaconSalaryInfo struct {
	BeaconSalary      uint64
	PayToAddress      *privacy.PaymentAddress
	BeaconBlockHeight uint64
	InfoHash          *common.Hash
}

func (beaconSalaryInfo *BeaconSalaryInfo) hash() *common.Hash {
	record := string(beaconSalaryInfo.BeaconSalary) + string(beaconSalaryInfo.BeaconBlockHeight)
	record += beaconSalaryInfo.PayToAddress.String()
	hash := common.HashH([]byte(record))
	return &hash
}

func BuildInstForBeaconSalary(salary, beaconHeight uint64, payToAddress *privacy.PaymentAddress) ([]string, error) {

	beaconSalaryInfo := BeaconSalaryInfo{
		BeaconBlockHeight: beaconHeight,
		PayToAddress:      payToAddress,
		BeaconSalary:      salary,
	}
	//fmt.Println("SA: beaconSalaryInfo", beaconSalaryInfo)
	beaconSalaryInfo.InfoHash = beaconSalaryInfo.hash()

	contentStr, err := json.Marshal(beaconSalaryInfo)
	if err != nil {
		return nil, err
	}

	returnedInst := []string{
		strconv.Itoa(BeaconSalaryRequestMeta),
		strconv.Itoa(int(common.GetShardIDFromLastByte(payToAddress.Bytes()[len(payToAddress.Bytes())-1]))),
		"beaconSalaryInst",
		string(contentStr),
	}

	return returnedInst, nil
}

type BeaconBlockSalaryRes struct {
	MetadataBase
	BeaconBlockHeight uint64
	ProducerAddress   *privacy.PaymentAddress
	InfoHash          *common.Hash
}

type BeaconBlockSalaryInfo struct {
	BeaconSalary      uint64
	PayToAddress      *privacy.PaymentAddress
	BeaconBlockHeight uint64
	InfoHash          *common.Hash
}

func NewBeaconBlockSalaryRes(
	beaconBlockHeight uint64,
	producerAddress *privacy.PaymentAddress,
	infoHash *common.Hash,
	metaType int,
) *BeaconBlockSalaryRes {
	metadataBase := MetadataBase{
		Type: metaType,
	}
	return &BeaconBlockSalaryRes{
		BeaconBlockHeight: beaconBlockHeight,
		ProducerAddress:   producerAddress,
		InfoHash:          infoHash,
		MetadataBase:      metadataBase,
	}
}

func (sbsRes *BeaconBlockSalaryRes) CheckTransactionFee(tr Transaction, minFee uint64) bool {
	// no need to have fee for this tx
	return true
}

func (sbsRes *BeaconBlockSalaryRes) ValidateTxWithBlockChain(txr Transaction, bcr BlockchainRetriever, shardID byte, db database.DatabaseInterface) (bool, error) {
	// no need to validate tx with blockchain, just need to validate with request tx (via RequestedTxID) in current block
	return false, nil
}

func (sbsRes *BeaconBlockSalaryRes) ValidateSanityData(bcr BlockchainRetriever, txr Transaction) (bool, bool, error) {
	if len(sbsRes.ProducerAddress.Pk) == 0 {
		return false, false, errors.New("Wrong request info's producer address")
	}
	if len(sbsRes.ProducerAddress.Tk) == 0 {
		return false, false, errors.New("Wrong request info's producer address")
	}
	// if sbsRes.ShardBlockHeight == 0 {
	// 	return false, false, errors.New("Wrong request info's shard block height")
	// }
	return false, true, nil
}

func (sbsRes *BeaconBlockSalaryRes) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return true
}

func (sbsRes *BeaconBlockSalaryRes) Hash() *common.Hash {
	record := sbsRes.ProducerAddress.String()
	record += string(sbsRes.BeaconBlockHeight)
	record += sbsRes.InfoHash.String()

	// final hash
	record += sbsRes.MetadataBase.Hash().String()
	hash := common.HashH([]byte(record))
	return &hash
}

func (sbsRes *BeaconBlockSalaryRes) VerifyMinerCreatedTxBeforeGettingInBlock(
	txsInBlock []Transaction,
	txsUsed []int,
	insts [][]string,
	instUsed []int,
	shardID byte,
	tx Transaction,
	bcr BlockchainRetriever,
) (bool, error) {
	instIdx := -1
	var beaconSalaryInfo BeaconSalaryInfo
	for i, inst := range insts {
		if instUsed[i] > 0 {
			continue
		}
		if inst[0] != strconv.Itoa(BeaconSalaryRequestMeta) {
			continue
		}
		if inst[1] != strconv.Itoa(int(shardID)) {
			continue
		}
		if inst[2] != "beaconSalaryInst" {
			continue
		}
		contentStr := inst[3]
		err := json.Unmarshal([]byte(contentStr), &beaconSalaryInfo)
		if err != nil {
			return false, err
		}

		if !bytes.Equal(beaconSalaryInfo.InfoHash[:], sbsRes.InfoHash[:]) {
			continue
		}
		instIdx = i
		instUsed[i] += 1
		break
	}
	if instIdx == -1 {
		return false, errors.Errorf("no instruction found for BeaconBlockSalaryResponse tx %s", tx.Hash().String())
	}
	if (!bytes.Equal(beaconSalaryInfo.PayToAddress.Pk[:], sbsRes.ProducerAddress.Pk[:])) ||
		(!bytes.Equal(beaconSalaryInfo.PayToAddress.Tk[:], sbsRes.ProducerAddress.Tk[:])) {
		return false, errors.Errorf("Producer address in BeaconBlockSalaryResponse tx %s is not matched to instruction's", tx.Hash().String())
	}
	if beaconSalaryInfo.BeaconBlockHeight != sbsRes.BeaconBlockHeight {
		return false, errors.Errorf("ShardBlockHeight in BeaconBlockSalaryResponse tx %s is not matched to instruction's", tx.Hash().String())
	}

	if beaconSalaryInfo.BeaconSalary != tx.CalculateTxValue() {
		//fmt.Println("SA: beacon salary info", beaconSalaryInfo)
		return false, errors.Errorf("Salary amount in BeaconBlockSalaryResponse tx %s is not matched to instruction's %d %d", tx.Hash().String(), beaconSalaryInfo.BeaconSalary, tx.CalculateTxValue())
	}

	return true, nil
}
