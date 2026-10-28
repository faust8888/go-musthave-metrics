package main

import (
	"strings"
	"testing"
)

func TestAnalyzersIncludeRequired(t *testing.T) {
	names := make(map[string]struct{})
	for _, a := range analyzers() {
		if a == nil || a.Name == "" {
			t.Fatal("nil or unnamed analyzer")
		}
		names[a.Name] = struct{}{}
	}

	required := []string{
		"printf",
		"exitcheck",
		"ineffassign",
		"bodyclose",
		"S1012",
		"SA4006",
		"SA1019",
		"SA5008",
	}
	for _, name := range required {
		if _, ok := names[name]; !ok {
			t.Errorf("missing analyzer %s", name)
		}
	}

	var sa int
	for name := range names {
		if strings.HasPrefix(name, "SA") {
			sa++
		}
	}
	if sa < 40 {
		t.Errorf("expected all SA analyzers, got %d", sa)
	}
}
