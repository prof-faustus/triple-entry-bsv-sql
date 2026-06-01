package cryptocore

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// AES256GCMEncrypt returns ciphertext ‖ — and the 16-byte tag separately. ALGORITHMS.md §5.
func AES256GCMEncrypt(key, nonce, aad, plaintext []byte) (ciphertext, tag []byte, err error) {
	if len(key) != 32 {
		return nil, nil, errors.New("AES-256-GCM key must be 32 bytes")
	}
	if len(nonce) != 12 {
		return nil, nil, errors.New("GCM nonce must be 12 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	sealed := g.Seal(nil, nonce, plaintext, aad)
	// Go appends the tag to the ciphertext; split it to match the TS/§5 layout.
	ct := sealed[:len(sealed)-g.Overhead()]
	t := sealed[len(sealed)-g.Overhead():]
	return ct, t, nil
}

func AES256GCMDecrypt(key, nonce, aad, ciphertext, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	sealed := append(append([]byte{}, ciphertext...), tag...)
	return g.Open(nil, nonce, sealed, aad)
}
