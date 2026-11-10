package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_EmptyPath(t *testing.T) {
	var f ServerFile
	if err := Load("", &f); err != nil {
		t.Fatalf("empty path: %v", err)
	}
	if f.Address != nil {
		t.Fatal("empty path must not fill fields")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	var f ServerFile
	if err := Load(filepath.Join(t.TempDir(), "nope.json"), &f); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	var f ServerFile
	if err := Load(path, &f); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoad_ServerFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.json")
	data := `{
		"address": "127.0.0.1:9090",
		"restore": false,
		"store_interval": "1s",
		"store_file": "/tmp/file.db",
		"database_dsn": "postgres://localhost/db",
		"crypto_key": "/tmp/key.pem",
		"key": "secret",
		"audit_file": "/tmp/audit.log",
		"audit_url": "http://audit"
	}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	var f ServerFile
	if err := Load(path, &f); err != nil {
		t.Fatal(err)
	}
	if f.Address == nil || *f.Address != "127.0.0.1:9090" {
		t.Errorf("address = %v", f.Address)
	}
	if f.Restore == nil || *f.Restore {
		t.Errorf("restore = %v", f.Restore)
	}
	if f.StoreInterval == nil || int(*f.StoreInterval) != 1 {
		t.Errorf("store_interval = %v", f.StoreInterval)
	}
	if f.StoreFile == nil || *f.StoreFile != "/tmp/file.db" {
		t.Errorf("store_file = %v", f.StoreFile)
	}
	if f.DatabaseDSN == nil || *f.DatabaseDSN != "postgres://localhost/db" {
		t.Errorf("database_dsn = %v", f.DatabaseDSN)
	}
	if f.CryptoKey == nil || *f.CryptoKey != "/tmp/key.pem" {
		t.Errorf("crypto_key = %v", f.CryptoKey)
	}
	if f.Key == nil || *f.Key != "secret" {
		t.Errorf("key = %v", f.Key)
	}
	if f.AuditFile == nil || *f.AuditFile != "/tmp/audit.log" {
		t.Errorf("audit_file = %v", f.AuditFile)
	}
	if f.AuditURL == nil || *f.AuditURL != "http://audit" {
		t.Errorf("audit_url = %v", f.AuditURL)
	}
}

func TestLoad_AgentFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	data := `{
		"address": "localhost:8081",
		"report_interval": "10s",
		"poll_interval": "2s",
		"crypto_key": "/tmp/pub.pem",
		"key": "k",
		"rate_limit": 4
	}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	var f AgentFile
	if err := Load(path, &f); err != nil {
		t.Fatal(err)
	}
	if f.Address == nil || *f.Address != "localhost:8081" {
		t.Errorf("address = %v", f.Address)
	}
	if f.ReportInterval == nil || int(*f.ReportInterval) != 10 {
		t.Errorf("report_interval = %v", f.ReportInterval)
	}
	if f.PollInterval == nil || int(*f.PollInterval) != 2 {
		t.Errorf("poll_interval = %v", f.PollInterval)
	}
	if f.RateLimit == nil || *f.RateLimit != 4 {
		t.Errorf("rate_limit = %v", f.RateLimit)
	}
}

func TestDurationSec_Unmarshal(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{`"1s"`, 1},
		{`"10s"`, 10},
		{`"2m"`, 120},
		{`5`, 5},
		{`"7"`, 7},
		{`null`, 0},
	}
	for _, tt := range tests {
		var d DurationSec
		if err := json.Unmarshal([]byte(tt.in), &d); err != nil {
			t.Errorf("Unmarshal(%s): %v", tt.in, err)
			continue
		}
		if int(d) != tt.want {
			t.Errorf("Unmarshal(%s) = %d, want %d", tt.in, d, tt.want)
		}
	}

	var bad DurationSec
	if err := json.Unmarshal([]byte(`"nope"`), &bad); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestDurationSec_IntPtr(t *testing.T) {
	if (*DurationSec)(nil).IntPtr() != nil {
		t.Fatal("nil receiver")
	}
	d := DurationSec(3)
	p := d.IntPtr()
	if p == nil || *p != 3 {
		t.Errorf("got %v", p)
	}
}

func TestApply(t *testing.T) {
	src := "from-file"
	dst := "default"
	Apply(&dst, &src, map[string]bool{}, "a")
	if dst != "from-file" {
		t.Errorf("got %q", dst)
	}

	dst = "flag-value"
	Apply(&dst, &src, map[string]bool{"a": true}, "a")
	if dst != "flag-value" {
		t.Errorf("flag must win, got %q", dst)
	}

	Apply(&dst, (*string)(nil), map[string]bool{}, "a")
	if dst != "flag-value" {
		t.Errorf("nil src must not change dest, got %q", dst)
	}
}

func TestVisitedSet(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = fs.String("a", "default", "")
	_ = fs.String("c", "", "")
	if err := fs.Parse([]string{"-a", "localhost:1"}); err != nil {
		t.Fatal(err)
	}
	got := VisitedSet(fs)
	if !got["a"] {
		t.Fatal("expected -a visited")
	}
	if got["c"] {
		t.Fatal("did not expect -c visited")
	}
}
