// Package config loads JSON configuration files for the agent and the server.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// ServerFile is the JSON configuration overlay for the server.
type ServerFile struct {
	Address       *string      `json:"address"`
	Restore       *bool        `json:"restore"`
	StoreInterval *DurationSec `json:"store_interval"`
	StoreFile     *string      `json:"store_file"`
	DatabaseDSN   *string      `json:"database_dsn"`
	CryptoKey     *string      `json:"crypto_key"`
	Key           *string      `json:"key"`
	AuditFile     *string      `json:"audit_file"`
	AuditURL      *string      `json:"audit_url"`
}

// AgentFile is the JSON configuration overlay for the agent.
type AgentFile struct {
	Address        *string      `json:"address"`
	ReportInterval *DurationSec `json:"report_interval"`
	PollInterval   *DurationSec `json:"poll_interval"`
	CryptoKey      *string      `json:"crypto_key"`
	Key            *string      `json:"key"`
	RateLimit      *int         `json:"rate_limit"`
}

// Load reads JSON from path into dest. An empty path is a no-op.
func Load(path string, dest any) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	return nil
}

// Visited reports which flags were explicitly set on the command line.
func Visited() map[string]bool {
	return VisitedSet(flag.CommandLine)
}

// VisitedSet reports which flags were explicitly set on fs.
func VisitedSet(fs *flag.FlagSet) map[string]bool {
	m := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		m[f.Name] = true
	})
	return m
}

// Apply copies *src into *dst unless src is nil or one of the named flags was set.
func Apply[T any](dst *T, src *T, visited map[string]bool, flagNames ...string) {
	if src == nil {
		return
	}
	for _, name := range flagNames {
		if visited[name] {
			return
		}
	}
	*dst = *src
}
