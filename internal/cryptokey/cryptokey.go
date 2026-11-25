// Package cryptokey loads RSA PEM keys and encrypts HTTP payloads with a
// hybrid RSA-OAEP + AES-GCM scheme so bodies larger than one RSA block
// can still be protected.
package cryptokey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

const (
	// Header is set on encrypted requests so the server can tell ciphertext
	// from plain JSON or gzip.
	Header = "Content-Encryption"
	// HeaderValue identifies the hybrid RSA-OAEP + AES-256-GCM envelope.
	HeaderValue = "rsa-aes-gcm"

	aesKeySize = 32
)

// LoadPublic reads a PEM-encoded PKIX or PKCS#1 RSA public key from path.
func LoadPublic(path string) (*rsa.PublicKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		pub, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("crypto-key: not an RSA public key")
		}
		return pub, nil
	}
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto-key: parse public key: %w", err)
	}
	return pub, nil
}

// LoadPrivate reads a PEM-encoded PKCS#1 or PKCS#8 RSA private key from path.
func LoadPrivate(path string) (*rsa.PrivateKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto-key: parse private key: %w", err)
	}
	priv, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("crypto-key: not an RSA private key")
	}
	return priv, nil
}

// Encrypt seals plaintext with a random AES-256-GCM key and wraps that key
// in RSA-OAEP. The envelope is rsa(key) || nonce || ciphertext.
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	aesKey := make([]byte, aesKeySize)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("crypto-key: aes key: %w", err)
	}
	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto-key: rsa wrap: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto-key: nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, len(wrapped)+len(nonce)+len(sealed))
	out = append(out, wrapped...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

// Decrypt opens an envelope produced by Encrypt.
func Decrypt(priv *rsa.PrivateKey, envelope []byte) ([]byte, error) {
	rsaLen := priv.Size()
	if len(envelope) < rsaLen {
		return nil, errors.New("crypto-key: ciphertext too short")
	}
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, envelope[:rsaLen], nil)
	if err != nil {
		return nil, fmt.Errorf("crypto-key: rsa unwrap: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	rest := envelope[rsaLen:]
	if len(rest) < gcm.NonceSize() {
		return nil, errors.New("crypto-key: ciphertext too short")
	}
	nonce, sealed := rest[:gcm.NonceSize()], rest[gcm.NonceSize():]
	return gcm.Open(nil, nonce, sealed, nil)
}

func readPEM(path string) (*pem.Block, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("crypto-key: read %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("crypto-key: no PEM block in %s", path)
	}
	return block, nil
}
