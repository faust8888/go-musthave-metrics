package cryptokey

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	priv := mustKey(t)
	plain := bytes.Repeat([]byte("metrics-payload-"), 50)
	enc, err := Encrypt(&priv.PublicKey, plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(enc, plain) {
		t.Fatal("ciphertext must differ from plaintext")
	}
	got, err := Decrypt(priv, enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("roundtrip mismatch: got %d bytes, want %d", len(got), len(plain))
	}
}

func TestLoadPEMKeys(t *testing.T) {
	priv := mustKey(t)
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "key.pub")
	privPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(pubPath, publicPEM(&priv.PublicKey), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privPath, privatePEM(priv), 0o600); err != nil {
		t.Fatal(err)
	}

	pub, err := LoadPublic(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPrivate(privPath)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("hello")
	enc, err := Encrypt(pub, msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(loaded, enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("got %q", got)
	}
}

func TestDecryptShort(t *testing.T) {
	priv := mustKey(t)
	if _, err := Decrypt(priv, []byte("nope")); err == nil {
		t.Fatal("want error for short ciphertext")
	}
}

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return priv
}

func publicPEM(pub *rsa.PublicKey) []byte {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}

func privatePEM(priv *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
}
