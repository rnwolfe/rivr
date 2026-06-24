// Command rivr is an agent-focused CLI for search and data retrieval against Amazon
// Shopping. It implements the agent-CLI contract (read-only by default, --json,
// schema --json, structured errors, bounded output, embedded SKILL.md) over a pluggable
// provider backend.
//
// Scaffolded by the agent-cli-factory from spec.md. The real provider integrations and
// auth (keyring) are wired by cli-implement; this skeleton ships a stub provider so the
// contract surface is provably correct and testable offline.
package main

import (
	"os"

	"github.com/rnwolfe/rivr/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
