package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type AES struct {
	Key []byte
}

func (aesObj *AES) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesObj.Key)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))

	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

func (aesObj *AES) decrypt(ciphertext []byte) ([]byte, error) {
	plaintext := make([]byte, len(ciphertext[aes.BlockSize:]))

	block, err := aes.NewCipher(aesObj.Key)
	if err != nil {
		return nil, err
	}

	iv := ciphertext[:aes.BlockSize]
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext[aes.BlockSize:])

	return plaintext, nil
}
