package hash

import (
	"testing"
)

func TestSign_Deterministic(t *testing.T) {
	data := []byte(`{"id":"Alloc","type":"gauge","value":1.5}`)
	key := "secret"

	a := Sign(data, key)
	b := Sign(data, key)
	if a == "" {
		t.Fatal("Sign returned empty string")
	}
	if a != b {
		t.Errorf("Sign is not deterministic: %q vs %q", a, b)
	}
}

func TestSign_DependsOnKeyAndData(t *testing.T) {
	data := []byte("payload")
	if Sign(data, "k1") == Sign(data, "k2") {
		t.Error("different keys produced the same hash")
	}
	if Sign([]byte("a"), "k") == Sign([]byte("b"), "k") {
		t.Error("different data produced the same hash")
	}
}

func TestEqual(t *testing.T) {
	data := []byte("hello")
	key := "key"

	if !Equal(Sign(data, key), data, key) {
		t.Error("Equal rejected a valid signature")
	}
	if Equal("deadbeef", data, key) {
		t.Error("Equal accepted an invalid signature")
	}
	if Equal(Sign(data, "other"), data, key) {
		t.Error("Equal accepted a signature made with another key")
	}
}
