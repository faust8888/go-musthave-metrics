// Package buildinfo prints binary version metadata to stdout.
package buildinfo

import "fmt"

// Print writes build version, date and commit in the format expected by the
// metrics service autotests. Callers pass the values of main.buildVersion,
// main.buildDate and main.buildCommit (defaults and ldflags live there).
func Print(version, date, commit string) {
	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}
