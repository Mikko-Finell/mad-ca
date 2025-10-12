//go:build !ebiten

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "The GUI build of mad-ca requires the ebiten build tag.")
	fmt.Fprintln(os.Stderr, "Re-run with `go run -tags ebiten ./cmd/ca` or build with `-tags ebiten`.")
	os.Exit(2)
}
