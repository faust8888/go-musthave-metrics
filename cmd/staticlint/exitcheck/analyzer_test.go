package exitcheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/faust8888/go-musthave-metrics/cmd/staticlint/exitcheck"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), exitcheck.Analyzer, "a", "b", "c")
}
