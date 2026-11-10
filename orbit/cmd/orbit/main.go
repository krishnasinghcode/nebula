// Package main provides the orbit CLI: serve (control plane API) and version.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{Use: "orbit", Short: "Nebula Orbit — local cloud compute"}
	root.AddCommand(newVersionCmd(), newServeCmd(), newKeyCmd(), newSSHCmd(),
		newInstanceCmd(), newImagesCmd(), newTypesCmd(), newCapacityCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the orbit version",
		Run:   func(*cobra.Command, []string) { fmt.Println("orbit " + version) },
	}
}
