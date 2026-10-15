package envconfig

import (
	"testing"
)

func TestString(t *testing.T) {
	t.Setenv("TEST_STR", "from-env")
	got := "default"
	String("TEST_STR", &got)
	if got != "from-env" {
		t.Errorf("got %q, want from-env", got)
	}

	empty := "keep"
	String("TEST_STR_MISSING", &empty)
	if empty != "keep" {
		t.Errorf("empty env must not override, got %q", empty)
	}
}

func TestInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	n := 1
	if err := Int("TEST_INT", &n); err != nil || n != 42 {
		t.Errorf("got n=%d err=%v, want 42 nil", n, err)
	}

	if err := Int("TEST_INT_MISSING", &n); err != nil || n != 42 {
		t.Errorf("missing env: n=%d err=%v", n, err)
	}

	t.Setenv("TEST_INT_BAD", "nope")
	if err := Int("TEST_INT_BAD", &n); err == nil {
		t.Fatal("expected error for invalid int")
	}
}

func TestBool(t *testing.T) {
	t.Setenv("TEST_BOOL", "true")
	b := false
	if err := Bool("TEST_BOOL", &b); err != nil || !b {
		t.Errorf("got b=%v err=%v, want true nil", b, err)
	}

	if err := Bool("TEST_BOOL_MISSING", &b); err != nil || !b {
		t.Errorf("missing env: b=%v err=%v", b, err)
	}

	t.Setenv("TEST_BOOL_BAD", "maybe")
	if err := Bool("TEST_BOOL_BAD", &b); err == nil {
		t.Fatal("expected error for invalid bool")
	}
}
