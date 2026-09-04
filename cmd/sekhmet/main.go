// Command sekhmet is the QYVORA high-performance fuzzing and
// vulnerability-discovery framework. It never calls os.Exit itself; the
// exit-code contract lives in the cli package so test binaries can exercise
// exit behavior directly.
package main

import (
	"os"

	"github.com/QYVORA/qyvora-sekhmet/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
