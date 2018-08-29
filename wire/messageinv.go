package wire

import (
	"encoding/hex"
	"encoding/json"
)

type MessageInv struct {
	InvList []InvVect
}

func (self MessageInv) MessageType() string {
	return CmdInv
}

func (self MessageInv) MaxPayloadLength(pver int) int {
	return MaxBlockPayload
}

func (self MessageInv) JsonSerialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(self)
	return jsonBytes, err
}

func (self MessageInv) JsonDeserialize(jsonStr string) error {
	jsonDecodeString, _ := hex.DecodeString(jsonStr)
	err := json.Unmarshal([]byte(jsonDecodeString), self)
	return err
}