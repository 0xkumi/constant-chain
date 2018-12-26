package zkp

import (
	"github.com/ninjadotorg/constant/common"
	"math"
	"math/big"

	"github.com/ninjadotorg/constant/privacy"
)

// GenerateChallengeFromPoint get hash of n points in G append with input values
// return blake_2b(G[0]||G[1]||...||G[CM_CAPACITY-1]||<values>)
// G[i] is list of all generator point of Curve
func GenerateChallengeFromPoint(values []*privacy.EllipticPoint) *big.Int {
	bytes := privacy.PedCom.G[0].Compress()
	for i := 1; i < privacy.PedCom.Capacity; i++ {
		bytes = append(bytes, privacy.PedCom.G[i].Compress()...)
	}

	for i := 0; i < len(values); i++ {
		bytes = append(bytes, values[i].Compress()...)
	}

	hash := common.HashB(bytes)

	res := new(big.Int).SetBytes(hash)
	res.Mod(res, privacy.Curve.Params().N)
	return res
}

// GenerateChallengeFromByte get hash of n points in G append with input values
// return blake_2b(G[0]||G[1]||...||G[CM_CAPACITY-1]||<values>)
// G[i] is list of all generator point of Curve
func GenerateChallengeFromByte(values [][]byte) *big.Int {
	bytes := privacy.PedCom.G[0].Compress()
	for i := 1; i < privacy.PedCom.Capacity; i++ {
		bytes = append(bytes, privacy.PedCom.G[i].Compress()...)
	}

	for i := 0; i < len(values); i++ {
		bytes = append(bytes, values[i]...)
	}

	hash := common.HashB(bytes)

	res := new(big.Int).SetBytes(hash)
	res.Mod(res, privacy.Curve.Params().N)
	return res
}

// EstimateProofSize returns the estimated size of the proof in kilobyte
func EstimateProofSize(nInput int, nOutput int) uint64 {
	sizeOneOfManyProof := nInput * privacy.OneOfManyProofSize
	sizeSNPrivacyProof := nInput * privacy.SNPrivacyProofSize

	sizeComOutputMultiRangeProof := int(EstimateMultiRangeProof(nOutput))
	sizeSumOutRangeProof := privacy.SumOutRangeProofSize
	sizeComZeroProof := privacy.ComZeroProofSize

	sizeInputCoins := nInput * privacy.InputCoinsPrivacySize
	sizeOutputCoins := nOutput * privacy.OutputCoinsPrivacySize

	sizeComOutputValue := nOutput * privacy.CompressedPointSize
	sizeComOutputSND := nOutput * privacy.CompressedPointSize
	sizeComOutputShardID := nOutput * privacy.CompressedPointSize

	sizeComInputSK := nInput * privacy.CompressedPointSize
	sizeComInputValue := nInput * privacy.CompressedPointSize
	sizeComInputSND := nInput * privacy.CompressedPointSize
	sizeComInputShardID := nInput * privacy.CompressedPointSize

	// sizeBytes = NumArr + SizeProof
	sizeBytes := 11 + 9*nInput + 4*nOutput + 4

	sizeProof := sizeOneOfManyProof + sizeSNPrivacyProof +
		sizeComOutputMultiRangeProof + sizeSumOutRangeProof + sizeComZeroProof + sizeInputCoins + sizeOutputCoins +
		sizeComOutputValue + sizeComOutputSND + sizeComOutputShardID + sizeComInputSK + sizeComInputValue + sizeComInputSND + sizeComInputShardID + sizeBytes

	return uint64(math.Ceil(float64(sizeProof) / 1024))
}

func EstimateMultiRangeProof(nOutput int) uint64 {
	sizeCounter := uint64(1)                                        // byte
	sizeComms := uint64(pad(nOutput) * privacy.CompressedPointSize) //  []*privacy.EllipticPoint
	sizeA := uint64(privacy.CompressedPointSize)                    //    *privacy.EllipticPoint
	sizeS := uint64(privacy.CompressedPointSize)                    //       *privacy.EllipticPoint
	sizeT1 := uint64(privacy.CompressedPointSize)                   //    *privacy.EllipticPoint
	sizeT2 := uint64(privacy.CompressedPointSize)                   //       *privacy.EllipticPoint

	sizeTau := uint64(privacy.BigIntSize) //    *big.Int
	sizeTh := uint64(privacy.BigIntSize)  //    *big.Int
	sizeMu := uint64(privacy.BigIntSize)  //    *big.Int

	a := privacy.MaxExp * pad(nOutput)
	a = int(math.Log2(float64(a)))
	sizeIPP := uint64(a*privacy.CompressedPointSize + a*privacy.CompressedPointSize + 2*privacy.BigIntSize + (a+1)*privacy.BigIntSize)

	sizeMaxExp := uint64(1)

	sizeCy := uint64(privacy.BigIntSize) //*big.Int
	sizeCz := uint64(privacy.BigIntSize) //*big.Int
	sizeCx := uint64(privacy.BigIntSize) //*big.Int

	return uint64(sizeCounter + sizeComms + sizeA + sizeS + sizeT1 + sizeT2 + sizeTau + sizeTh + sizeMu + sizeIPP + sizeMaxExp + sizeCy + sizeCz + sizeCx)
}
